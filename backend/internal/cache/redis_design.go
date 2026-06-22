package cache

// RedisMode defines how the Redis adapter should be initialized.
// Real production implementation should use go-redis UniversalClient so one
// adapter supports both standalone Redis and Redis Cluster.
type RedisMode string

const (
	RedisModeSingle  RedisMode = "single"
	RedisModeCluster RedisMode = "cluster"
)

type RedisConfig struct {
	Mode     RedisMode `yaml:"mode" json:"mode"` // single / cluster
	Addrs    []string  `yaml:"addrs" json:"addrs"`
	Username string    `yaml:"username" json:"username"`
	Password string    `yaml:"password" json:"password"`
	DB       int       `yaml:"db" json:"db"` // single only; ignored by cluster
	Prefix   string    `yaml:"prefix" json:"prefix"`
}
