package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryCache is a process-local fallback for single-node deployments
// without Redis. Not shared across replicas; multiple server instances on
// MemoryCache will see stale RBAC reads.
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]memoryEntry
}

type memoryEntry struct {
	value   string
	expires time.Time
}

func NewMemory() *MemoryCache {
	c := &MemoryCache{data: make(map[string]memoryEntry)}
	go c.reaper()
	return c
}

func (c *MemoryCache) reaper() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.data {
			if !e.expires.IsZero() && now.After(e.expires) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *MemoryCache) Get(_ context.Context, key string) (string, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	if !ok {
		return "", false, nil
	}
	if !e.expires.IsZero() && time.Now().After(e.expires) {
		return "", false, nil
	}
	return e.value, true, nil
}

func (c *MemoryCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	c.data[key] = memoryEntry{value: value, expires: expires}
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

func (c *MemoryCache) DeletePrefix(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
		}
	}
	return nil
}

func (c *MemoryCache) Close() error { return nil }
