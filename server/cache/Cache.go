package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps Redis. Cached data is only ever used for display: anything that
// moves money reads Postgres.
type Cache struct {
	client *redis.Client
}

// Connect opens the Redis client and checks it's reachable.
func Connect(ctx context.Context, url string) (*Cache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}
	return &Cache{client: client}, nil
}

// Get loads key into dest and reports whether it was found. A Redis failure
// counts as a miss, so an outage slows the app down instead of breaking it.
func (c *Cache) Get(ctx context.Context, key string, dest any) bool {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			slog.Warn("cache get failed", "error", err)
		}
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

// Set stores value under key until ttl passes.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		slog.Warn("cache encode failed", "error", err)
		return
	}
	if err := c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		slog.Warn("cache set failed", "error", err)
	}
}

// Delete removes keys, used after a write makes them stale.
func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		slog.Warn("cache delete failed", "error", err)
	}
}

// Allow counts one hit against key and reports whether it's within limit for
// the current window. It returns an error if Redis is down, and callers must
// refuse the request rather than skip the limit.
func (c *Cache) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	pipe := c.client.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return count.Val() <= limit, nil
}

// Ping reports whether Redis is reachable.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close releases the Redis connections.
func (c *Cache) Close() error {
	return c.client.Close()
}
