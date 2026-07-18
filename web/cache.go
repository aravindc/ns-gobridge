package web

import (
	"sync"
	"time"
)

// ttlCache is a minimal in-process cache for read-heavy endpoints polled by
// many clients (e.g. IoT devices). It avoids re-querying Postgres on every
// request within the TTL window, regardless of how many clients are polling.
type ttlCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]cacheEntry
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

func newTTLCache(ttl time.Duration) *ttlCache {
	return &ttlCache{ttl: ttl, entries: make(map[string]cacheEntry)}
}

func (c *ttlCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *ttlCache) set(key string, value any) {
	c.setWithTTL(key, value, c.ttl)
}

// setWithTTL stores value under key with a TTL different from the cache's
// default, e.g. for entries that are expensive to compute and change slowly.
func (c *ttlCache) setWithTTL(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}
