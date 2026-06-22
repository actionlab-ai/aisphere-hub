package redislock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Mode               string
	Addrs              []string
	Username, Password string
	DB                 int
	Prefix             string
}

type LockManager struct {
	client redis.UniversalClient
	prefix string
}

func New(cfg Config) (*LockManager, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"127.0.0.1:6379"}
	}
	u := &redis.UniversalOptions{Addrs: cfg.Addrs, Username: cfg.Username, Password: cfg.Password, DB: cfg.DB}
	var cli redis.UniversalClient
	if strings.EqualFold(cfg.Mode, "cluster") {
		cli = redis.NewClusterClient(u.Cluster())
	} else {
		cli = redis.NewClient(u.Simple())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Ping(ctx).Err(); err != nil {
		_ = cli.Close()
		return nil, err
	}
	return &LockManager{client: cli, prefix: strings.Trim(cfg.Prefix, ":")}, nil
}

func (m *LockManager) k(k string) string {
	if m.prefix == "" {
		return k
	}
	return m.prefix + ":" + k
}

func (m *LockManager) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	token := randToken()
	redisKey := m.k(key)
	acquireDeadline := time.Now().Add(ttl)
	for {
		ok, err := m.client.SetNX(ctx, redisKey, token, ttl).Result()
		if err != nil {
			return err
		}
		if ok {
			break
		}
		if time.Now().After(acquireDeadline) {
			return fmt.Errorf("acquire lock timeout: %s", key)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
		}
	}
	defer m.unlock(context.Background(), redisKey, token)
	return fn()
}

func (m *LockManager) Wait(ctx context.Context, key string, maxWait time.Duration) error {
	if maxWait <= 0 {
		return nil
	}
	redisKey := m.k(key)
	deadline := time.Now().Add(maxWait)
	for {
		n, err := m.client.Exists(ctx, redisKey).Result()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait lock timeout: %s", key)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
		}
	}
}

func (m *LockManager) unlock(ctx context.Context, redisKey, token string) {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_ = m.client.Eval(ctx, script, []string{redisKey}, token).Err()
}

func randToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
