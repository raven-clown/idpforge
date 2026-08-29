package cache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedis(addr, password string, db int) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (r *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisCache) DeletePrefix(ctx context.Context, prefix string) error {
	iter := r.client.Scan(ctx, 0, prefix+"*", 500).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 500 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (r *RedisCache) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 && ttl > 0 {
		r.client.Expire(ctx, key, ttl)
	}
	return n, nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Publish and Subscribe back the realtime hub's cross-instance fan-out
// (see internal/httpserver/ws.go): with Redis enabled, an announcement or
// audit event posted on one instance reaches browsers connected to every
// other instance too, not just the one that handled the request. Not part
// of the Cache interface -- pub/sub across processes only means anything
// for Redis, never the in-memory single-node fallback.
func (r *RedisCache) Publish(ctx context.Context, channel string, message []byte) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe returns a channel of message payloads on the given Redis
// channel. The returned func must be called to release the subscription;
// it's safe to call more than once.
func (r *RedisCache) Subscribe(ctx context.Context, channel string) (<-chan []byte, func()) {
	sub := r.client.Subscribe(ctx, channel)
	out := make(chan []byte)
	go func() {
		defer close(out)
		for msg := range sub.Channel() {
			out <- []byte(msg.Payload)
		}
	}()
	var once sync.Once
	return out, func() { once.Do(func() { _ = sub.Close() }) }
}
