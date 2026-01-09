package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// Cache is a simple in-memory TTL cache.
type Cache[V any] struct {
	mu   sync.RWMutex
	data map[string]entry[V]
	ttl  time.Duration
}

// New creates a new Cache with the given TTL.
func New[V any](ttl time.Duration) *Cache[V] {
	return &Cache[V]{
		data: make(map[string]entry[V]),
		ttl:  ttl,
	}
}

// Get returns the cached value for the given key, if it exists and is not expired.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var zero V

	e, ok := c.data[key]
	if !ok {
		return zero, false
	}
	if time.Now().After(e.expiresAt) {
		return zero, false
	}
	return e.value, true
}

// Set stores a value in the cache.
func (c *Cache[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = entry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes all entries from the cache.
func (c *Cache[V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]entry[V])
}
