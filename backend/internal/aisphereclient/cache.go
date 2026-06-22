package aisphereclient

import (
	"errors"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth"
)

// ErrDisabled is returned when the aisphere-auth integration is disabled.
// Providers translate this into "no identity / deny" rather than a hard
// error, so AIHub can run without aisphere-auth.
var ErrDisabled = errors.New("aisphere-auth integration disabled")

// introspectCache is a tiny TTL cache for introspect results. It exists
// because the SDK middleware layer at aisphere-auth already short-circuits
// /auth/me on its own, but AIHub APIs are typically hit in bursts by
// the same session cookie. A short (e.g. 5s) cache removes most of the
// per-request RPC cost without noticeably increasing staleness.
type introspectCache struct {
	ttl  time.Duration
	mu   sync.RWMutex
	data map[string]introspectCacheEntry
}

type introspectCacheEntry struct {
	principal *aisphereauth.Principal
	expiresAt time.Time
}

func newIntrospectCache(ttl time.Duration) *introspectCache {
	if ttl <= 0 {
		return nil
	}
	c := &introspectCache{ttl: ttl, data: map[string]introspectCacheEntry{}}
	go c.cleanupLoop(time.Minute)
	return c
}

func (c *introspectCache) get(sessionID string) (*aisphereauth.Principal, bool) {
	if c == nil || sessionID == "" {
		return nil, false
	}
	c.mu.RLock()
	entry, ok := c.data[sessionID]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	cp := *entry.principal
	return &cp, true
}

func (c *introspectCache) put(sessionID string, p *aisphereauth.Principal) {
	if c == nil || sessionID == "" || p == nil {
		return
	}
	cp := *p
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[sessionID] = introspectCacheEntry{principal: &cp, expiresAt: time.Now().Add(c.ttl)}
}

func (c *introspectCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.data {
			if now.After(v.expiresAt) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}
