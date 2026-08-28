// Package cache defines a small get/set/delete interface backed by Redis in
// production, with an in-memory fallback for single-node/dev deployments
// that don't want to run a separate cache server.
package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	// DeletePrefix removes every key starting with prefix; used to
	// invalidate all cached RBAC resolutions for a user/group/role at once.
	DeletePrefix(ctx context.Context, prefix string) error
	// Increment atomically increments key (creating it at 1 if absent),
	// sets ttl only on creation, and returns the new value. Used for
	// counter-based rate limiting.
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Close() error
}
