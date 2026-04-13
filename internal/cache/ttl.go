// Package cache provides a generic TTL-based in-memory cache.
package cache

import (
	"sync"
	"time"
)

type item struct {
	value    any
	expires  time.Time
	hasTTL   bool
}

// TTLCache is a tiny thread-safe in-memory cache.
type TTLCache struct {
	mu    sync.RWMutex
	items map[string]item
}

// New creates an empty cache.
func New() *TTLCache {
	return &TTLCache{
		items: make(map[string]item),
	}
}

// Set stores value with a TTL. ttl <= 0 means no expiration.
func (c *TTLCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := item{
		value:  value,
		hasTTL: ttl > 0,
	}
	if ttl > 0 {
		entry.expires = time.Now().Add(ttl)
	}
	c.items[key] = entry
}

// Get returns a cached value if present and not expired.
func (c *TTLCache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if entry.hasTTL && time.Now().After(entry.expires) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.value, true
}

