package repo

// MySQLConfig keeps database details outside service code. A production
// implementation can be done with database/sql + go-sql-driver/mysql, sqlc,
// GORM, or xorm; the upper layer only sees Repository interfaces.
type MySQLConfig struct {
	DSN             string `yaml:"dsn" json:"dsn"`
	MaxOpenConns    int    `yaml:"maxOpenConns" json:"maxOpenConns"`
	MaxIdleConns    int    `yaml:"maxIdleConns" json:"maxIdleConns"`
	ConnMaxLifetime string `yaml:"connMaxLifetime" json:"connMaxLifetime"`
}
