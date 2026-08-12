package metrics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "unicore_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	ResultsCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "unicore_results_cache_hits_total",
		Help: "Results cache hits",
	})
	ResultsCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "unicore_results_cache_misses_total",
		Help: "Results cache misses",
	})
	// Messages abandoned after exhausting their retries. Any non-zero rate here
	// is work that silently did not happen and needs a human.
	QueueDeadLetters = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "unicore_queue_dead_letters_total",
		Help: "Queue messages terminated after exhausting retries",
	}, []string{"topic"})
	// Outbox rows that stopped being retried by the relay, sampled each drain.
	OutboxDeadLetters = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "unicore_outbox_dead_letters",
		Help: "Outbox events that exhausted their delivery attempts",
	})
)

func Handler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}
