package cache

import (
	"context"
	"time"
)

type NoopCache struct{}

func (NoopCache) Get(context.Context, string) ([]byte, error)              { return nil, nil }
func (NoopCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (NoopCache) Delete(context.Context, ...string) error                  { return nil }
func (NoopCache) IncrBy(context.Context, string, int64) (int64, error)     { return 0, nil }
func (NoopCache) Expire(context.Context, string, time.Duration) error      { return nil }
