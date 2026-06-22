package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type MySQLConfig struct {
	DSN        string
	Dir        string
	AutoCreate bool
	Charset    string
	Collation  string
}

func RunMySQL(ctx context.Context, cfg MySQLConfig) error {
	if strings.TrimSpace(cfg.DSN) == "" {
		return fmt.Errorf("mysql dsn is required")
	}
	if strings.TrimSpace(cfg.Dir) == "" {
		cfg.Dir = "./migrations"
	}
	if cfg.Charset == "" {
		cfg.Charset = "utf8mb4"
	}
	if cfg.Collation == "" {
		cfg.Collation = "utf8mb4_unicode_ci"
	}
	if cfg.AutoCreate {
		if err := EnsureMySQLDatabase(ctx, cfg); err != nil {
			return err
		}
	}
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(cctx); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
        version VARCHAR(255) PRIMARY KEY,
        checksum VARCHAR(64),
        applied_at DATETIME NOT NULL
    )`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(cfg.Dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		version := filepath.Base(file)
		applied, err := isApplied(ctx, db, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if err := applySQL(ctx, db, string(b)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations(version, checksum, applied_at) VALUES(?,?,?)`, version, "", time.Now()); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	return nil
}

var safeIdent = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// EnsureMySQLDatabase connects to the MySQL server without selecting the target
// database, creates the database when it is missing, and then lets RunMySQL
// connect with the original DSN to apply table migrations.
func EnsureMySQLDatabase(ctx context.Context, cfg MySQLConfig) error {
	parsed, err := mysql.ParseDSN(cfg.DSN)
	if err != nil {
		return fmt.Errorf("parse mysql dsn: %w", err)
	}
	dbName := strings.TrimSpace(parsed.DBName)
	if dbName == "" {
		return fmt.Errorf("mysql dsn must include database name when autoCreate is enabled")
	}
	if !safeIdent.MatchString(dbName) {
		return fmt.Errorf("unsafe mysql database name %q; only letters, numbers and underscore are supported", dbName)
	}
	serverCfg := *parsed
	serverCfg.DBName = ""
	db, err := sql.Open("mysql", serverCfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(cctx); err != nil {
		return fmt.Errorf("connect mysql server for database bootstrap: %w", err)
	}
	charset := strings.TrimSpace(cfg.Charset)
	collation := strings.TrimSpace(cfg.Collation)
	if charset == "" {
		charset = "utf8mb4"
	}
	if collation == "" {
		collation = "utf8mb4_unicode_ci"
	}
	if !safeIdent.MatchString(charset) || !safeIdent.MatchString(collation) {
		return fmt.Errorf("unsafe mysql charset/collation")
	}
	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET %s COLLATE %s", dbName, charset, collation)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}

func isApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version=?`, version).Scan(&n)
	return n > 0, err
}

func applySQL(ctx context.Context, db *sql.DB, script string) error {
	stmts := splitStatements(script)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			if isIgnorableMigrationError(err) {
				continue
			}
			return err
		}
	}
	return tx.Commit()
}

func isIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	if me, ok := err.(*mysql.MySQLError); ok {
		// 1060 duplicate column, 1061 duplicate key name, 1091 cannot drop missing key/column.
		// We intentionally keep migration files simple and let the runner tolerate
		// additive DDL being re-applied across MySQL 5.7/8.x variants.
		return me.Number == 1060 || me.Number == 1061 || me.Number == 1091
	}
	return false
}

func splitStatements(script string) []string {
	lines := strings.Split(script, "\n")
	var b strings.Builder
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "--") || strings.HasPrefix(t, "#") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.Split(b.String(), ";")
}
