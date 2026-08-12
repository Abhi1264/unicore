package cache

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrUnavailable = errors.New("redis unavailable")

// Miss is redis.Nil for callers.
var Miss = redis.Nil

type Client struct {
	rdb *redis.Client
	log *slog.Logger
	// healthy is shared across request goroutines.
	healthy atomic.Bool
}

func New(ctx context.Context, redisURL string, log *slog.Logger) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	c := &Client{rdb: redis.NewClient(opts), log: log}
	c.healthy.Store(true)
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		log.Warn("redis ping failed; continuing with degraded cache", "error", err)
		c.healthy.Store(false)
	}
	return c, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Available() bool {
	return c.healthy.Load()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			c.markUp()
			return "", redis.Nil
		}
		c.markDown(err)
		return "", ErrUnavailable
	}
	c.markUp()
	return val, nil
}

func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		c.markDown(err)
		return ErrUnavailable
	}
	c.markUp()
	return nil
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		c.markDown(err)
		return ErrUnavailable
	}
	c.markUp()
	return nil
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		c.markDown(err)
		return 0, ErrUnavailable
	}
	c.markUp()
	return n, nil
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		c.markDown(err)
		return ErrUnavailable
	}
	c.markUp()
	return nil
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		c.markDown(err)
		return 0, ErrUnavailable
	}
	c.markUp()
	return d, nil
}

func (c *Client) markUp() {
	if c.healthy.CompareAndSwap(false, true) {
		c.log.Info("redis recovered")
	}
}

func (c *Client) markDown(err error) {
	if c.healthy.CompareAndSwap(true, false) {
		c.log.Warn("redis error; degrading", "error", err)
	}
}
