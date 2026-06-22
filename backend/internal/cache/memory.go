package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
)

type item struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu sync.RWMutex
	m  map[string]item
}

func NewMemoryCache() *MemoryCache { return &MemoryCache{m: map[string]item{}} }

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	it, ok := c.m[key]
	c.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		_ = c.Delete(ctx, key)
		return nil, nil
	}
	return append([]byte(nil), it.value...), nil
}
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	it := item{value: append([]byte(nil), value...)}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.m[key] = it
	c.mu.Unlock()
	return nil
}
func (c *MemoryCache) Delete(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		if strings.HasSuffix(key, "*") {
			prefix := strings.TrimSuffix(key, "*")
			for k := range c.m {
				if strings.HasPrefix(k, prefix) {
					delete(c.m, k)
				}
			}
			continue
		}
		delete(c.m, key)
	}
	return nil
}
func (c *MemoryCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var n int64
	if it, ok := c.m[key]; ok {
		_ = json.Unmarshal(it.value, &n)
	}
	n += delta
	b, _ := json.Marshal(n)
	c.m[key] = item{value: b}
	return n, nil
}
func (c *MemoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.m[key]
	if !ok {
		return nil
	}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	} else {
		it.expiresAt = time.Time{}
	}
	c.m[key] = it
	return nil
}

type Runtime struct{ raw ports.Cache }

func NewRuntime(raw ports.Cache) *Runtime {
	if raw == nil {
		raw = NewMemoryCache()
	}
	return &Runtime{raw: raw}
}

func (r *Runtime) GetRoute(ctx context.Context, ns, typ, name, label string) (string, bool, error) {
	b, err := r.raw.Get(ctx, KeyRoute(ns, typ, name, label))
	if err != nil || len(b) == 0 {
		return "", false, err
	}
	return string(b), true, nil
}
func (r *Runtime) SetRoute(ctx context.Context, ns, typ, name, label, version string, ttl time.Duration) error {
	return r.raw.Set(ctx, KeyRoute(ns, typ, name, label), []byte(version), ttl)
}
func (r *Runtime) DeleteRoutes(ctx context.Context, ns, typ, name string, labels ...string) error {
	if len(labels) == 0 {
		return r.raw.Delete(ctx, "aihub:route:"+typ+":"+name+":*")
	}
	keys := make([]string, 0, len(labels))
	for _, l := range labels {
		keys = append(keys, KeyRoute(ns, typ, name, l))
	}
	return r.raw.Delete(ctx, keys...)
}
func (r *Runtime) GetVersionMeta(ctx context.Context, ns, typ, name, version string, out any) (bool, error) {
	b, err := r.raw.Get(ctx, KeyVersion(ns, typ, name, version))
	if err != nil || len(b) == 0 {
		return false, err
	}
	return true, json.Unmarshal(b, out)
}
func (r *Runtime) SetVersionMeta(ctx context.Context, ns, typ, name, version string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.raw.Set(ctx, KeyVersion(ns, typ, name, version), b, ttl)
}
func (r *Runtime) DeleteVersionMeta(ctx context.Context, ns, typ, name string, versions ...string) error {
	if len(versions) == 0 {
		return r.raw.Delete(ctx, "aihub:version:"+typ+":"+name+":*")
	}
	keys := make([]string, 0, len(versions))
	for _, v := range versions {
		keys = append(keys, KeyVersion(ns, typ, name, v))
	}
	return r.raw.Delete(ctx, keys...)
}
func (r *Runtime) GetGroupManifest(ctx context.Context, ns, groupName, label string, out any) (bool, error) {
	b, err := r.raw.Get(ctx, KeyGroup(ns, groupName, label))
	if err != nil || len(b) == 0 {
		return false, err
	}
	return true, json.Unmarshal(b, out)
}
func (r *Runtime) SetGroupManifest(ctx context.Context, ns, groupName, label string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.raw.Set(ctx, KeyGroup(ns, groupName, label), b, ttl)
}
func (r *Runtime) DeleteGroupManifests(ctx context.Context, ns, groupName string) error {
	return r.raw.Delete(ctx, "aihub:skillset:"+groupName+":*")
}
func (r *Runtime) IncrementDownload(ctx context.Context, ns, typ, name, version string, delta int64) error {
	_, err := r.raw.IncrBy(ctx, KeyDownload(ns, typ, name, version), delta)
	return err
}

func KeyRoute(ns, typ, name, label string) string {
	return "aihub:route:" + typ + ":" + name + ":" + label
}
func KeyVersion(ns, typ, name, version string) string {
	return "aihub:version:" + typ + ":" + name + ":" + version
}
func KeyGroup(ns, groupName, label string) string {
	return "aihub:skillset:" + groupName + ":" + label
}
func KeyDownload(ns, typ, name, version string) string {
	return "aihub:download:" + typ + ":" + name + ":" + version
}
