package memorylock

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type LockManager struct {
	mu    sync.Mutex
	locks map[string]time.Time
}

func New() *LockManager { return &LockManager{locks: map[string]time.Time{}} }

func (m *LockManager) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	deadline := time.Now().Add(ttl)
	for {
		m.mu.Lock()
		exp, held := m.locks[key]
		if !held || time.Now().After(exp) {
			m.locks[key] = deadline
			m.mu.Unlock()
			defer func() { m.mu.Lock(); delete(m.locks, key); m.mu.Unlock() }()
			return fn()
		}
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func (m *LockManager) Wait(ctx context.Context, key string, maxWait time.Duration) error {
	if maxWait <= 0 {
		return nil
	}
	t := time.NewTimer(maxWait)
	defer t.Stop()
	for {
		m.mu.Lock()
		exp, held := m.locks[key]
		unlocked := !held || time.Now().After(exp)
		if held && time.Now().After(exp) {
			delete(m.locks, key)
		}
		m.mu.Unlock()
		if unlocked {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			return fmt.Errorf("wait lock timeout: %s", key)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
