package rediscache

import (
	"context"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Mode               string
	Addrs              []string
	Username, Password string
	DB                 int
	Prefix             string
}
type Cache struct {
	client redis.UniversalClient
	prefix string
}

func New(cfg Config) (*Cache, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"127.0.0.1:6379"}
	}
	mode := strings.ToLower(cfg.Mode)
	if mode == "" {
		mode = "single"
	}
	u := &redis.UniversalOptions{Addrs: cfg.Addrs, Username: cfg.Username, Password: cfg.Password, DB: cfg.DB}
	var cli redis.UniversalClient
	if mode == "cluster" {
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
	return &Cache{client: cli, prefix: strings.Trim(cfg.Prefix, ":")}, nil
}
func (c *Cache) k(k string) string {
	if c.prefix == "" {
		return k
	}
	return c.prefix + ":" + k
}
func (c *Cache) Close() error { return c.client.Close() }
func (c *Cache) Get(ctx context.Context, k string) ([]byte, error) {
	b, err := c.client.Get(ctx, c.k(k)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return b, err
}
func (c *Cache) Set(ctx context.Context, k string, v []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.k(k), v, ttl).Err()
}
func (c *Cache) IncrBy(ctx context.Context, k string, d int64) (int64, error) {
	return c.client.IncrBy(ctx, c.k(k), d).Result()
}
func (c *Cache) Expire(ctx context.Context, k string, ttl time.Duration) error {
	return c.client.Expire(ctx, c.k(k), ttl).Err()
}
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		if strings.HasSuffix(k, "*") {
			if err := c.deletePattern(ctx, c.k(k)); err != nil {
				return err
			}
			continue
		}
		if err := c.client.Del(ctx, c.k(k)).Err(); err != nil {
			return err
		}
	}
	return nil
}
func (c *Cache) deletePattern(ctx context.Context, pat string) error {
	if cc, ok := c.client.(*redis.ClusterClient); ok {
		return cc.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error { return scanDelete(ctx, shard, pat) })
	}
	if sc, ok := c.client.(*redis.Client); ok {
		return scanDelete(ctx, sc, pat)
	}
	var cur uint64
	for {
		ks, n, err := c.client.Scan(ctx, cur, pat, 200).Result()
		if err != nil {
			return err
		}
		if len(ks) > 0 {
			if err := c.client.Del(ctx, ks...).Err(); err != nil {
				return err
			}
		}
		cur = n
		if cur == 0 {
			return nil
		}
	}
}
func scanDelete(ctx context.Context, cli *redis.Client, pat string) error {
	var cur uint64
	for {
		ks, n, err := cli.Scan(ctx, cur, pat, 200).Result()
		if err != nil {
			return err
		}
		if len(ks) > 0 {
			if err := cli.Del(ctx, ks...).Err(); err != nil {
				return err
			}
		}
		cur = n
		if cur == 0 {
			return nil
		}
	}
}

var _ ports.Cache = (*Cache)(nil)
