package ports

import (
	"context"
	"time"
)

// LockManager is a resource-level coordination port. It is intentionally not a
// global Redis lock. Writers lock only the resource being changed, for example:
// aihub:lock:skill:{namespace}:{name}. Runtime readers can briefly wait on
// the same key before trusting cache-aside data.
type LockManager interface {
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error
	Wait(ctx context.Context, key string, maxWait time.Duration) error
}
