package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Abhi1264/unicore/api/internal/metrics"
	"github.com/nats-io/nats.go"
)

const (
	TopicPaymentConfirmed = "payments.confirmed"
	TopicPDFGenerate      = "documents.generate"
	TopicBulkImport       = "bulk.import"
	TopicDeadLetter       = "dead.letter"
)

const (
	// maxDeliveries must match len(retryBackoff) or NATS pads the backoff tail.
	maxDeliveries = 5
	ackWait       = 30 * time.Second

	streamMaxAge   = 7 * 24 * time.Hour
	streamMaxBytes = 1 << 30
)

var retryBackoff = []time.Duration{
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
}

type Client struct {
	nc  *nats.Conn
	js  nats.JetStreamContext
	log *slog.Logger
	ok  bool
}

func New(natsURL string, log *slog.Logger) (*Client, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Warn("nats connect failed; async features degraded", "error", err)
		return &Client{log: log, ok: false}, nil
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	c := &Client{nc: nc, js: js, log: log, ok: true}
	for _, stream := range []struct {
		name      string
		subjects  []string
		retention nats.RetentionPolicy
	}{
		{"PAYMENTS", []string{TopicPaymentConfirmed}, nats.WorkQueuePolicy},
		{"DOCUMENTS", []string{TopicPDFGenerate}, nats.WorkQueuePolicy},
		{"BULK", []string{TopicBulkImport}, nats.WorkQueuePolicy},
		{"DEAD_LETTER", []string{TopicDeadLetter}, nats.LimitsPolicy},
	} {
		cfg := &nats.StreamConfig{
			Name:      stream.name,
			Subjects:  stream.subjects,
			Retention: stream.retention,
			MaxAge:    streamMaxAge,
			MaxBytes:  streamMaxBytes,
		}
		if _, err := js.AddStream(cfg); err != nil {
			if _, updateErr := js.UpdateStream(cfg); updateErr != nil {
				log.Warn("could not reconcile stream config",
					"stream", stream.name, "add_error", err, "update_error", updateErr)
			}
		}
	}
	return c, nil
}

func (c *Client) Available() bool { return c.ok && c.nc != nil && c.nc.IsConnected() }

func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

func (c *Client) Publish(ctx context.Context, topic string, payload any) error {
	if !c.Available() {
		return fmt.Errorf("nats unavailable")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.js.Publish(topic, b, nats.Context(ctx))
	return err
}

// Subscribe registers a durable consumer; exhausted messages go to TopicDeadLetter then Term().
func (c *Client) Subscribe(topic, durable string, handler func(msg []byte) error) error {
	if !c.Available() {
		return fmt.Errorf("nats unavailable")
	}
	_, err := c.js.Subscribe(topic, func(m *nats.Msg) {
		err := runHandler(handler, m.Data)
		if err == nil {
			_ = m.Ack()
			return
		}

		attempt := deliveryAttempt(m)
		if attempt >= maxDeliveries {
			c.log.Error("queue message dead-lettered",
				"topic", topic, "durable", durable, "attempts", attempt, "error", err)
			metrics.QueueDeadLetters.WithLabelValues(topic).Inc()
			c.publishDeadLetter(topic, durable, attempt, m.Data, err)
			_ = m.Term()
			return
		}

		c.log.Warn("queue handler failed; will retry",
			"topic", topic, "durable", durable, "attempt", attempt, "error", err)
		_ = m.NakWithDelay(backoffFor(attempt))
	},
		nats.Durable(durable),
		nats.ManualAck(),
		nats.MaxDeliver(maxDeliveries),
		nats.AckWait(ackWait),
		nats.BackOff(retryBackoff),
	)
	return err
}

func runHandler(handler func([]byte) error, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return handler(data)
}

func (c *Client) publishDeadLetter(topic, durable string, attempt int, data []byte, cause error) {
	if !c.Available() {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"topic":     topic,
		"durable":   durable,
		"attempts":  attempt,
		"error":     cause.Error(),
		"payload":   json.RawMessage(data),
		"failed_at": time.Now().UTC(),
	})
	if err != nil {
		c.log.Error("marshal dead letter", "topic", topic, "error", err)
		return
	}
	if _, err := c.js.Publish(TopicDeadLetter, payload); err != nil {
		c.log.Error("publish dead letter", "topic", topic, "error", err)
	}
}

// deliveryAttempt is 1-based; missing JetStream metadata counts as attempt 1.
func deliveryAttempt(m *nats.Msg) int {
	meta, err := m.Metadata()
	if err != nil || meta == nil {
		return 1
	}
	return int(meta.NumDelivered)
}

func backoffFor(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > len(retryBackoff) {
		return retryBackoff[len(retryBackoff)-1]
	}
	return retryBackoff[attempt-1]
}
