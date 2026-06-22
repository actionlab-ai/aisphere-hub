package postgresstore

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var databaseNameRE = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

type databaseAdmin interface {
	databaseExists(context.Context, string) (bool, error)
	createDatabase(context.Context, string) error
}

type pgxDatabaseAdmin struct {
	conn *pgx.Conn
}

func (a pgxDatabaseAdmin) databaseExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := a.conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists)
	return exists, err
}

func (a pgxDatabaseAdmin) createDatabase(ctx context.Context, name string) error {
	_, err := a.conn.Exec(ctx, createDatabaseSQL(name))
	return err
}

func bootstrapDatabase(ctx context.Context, admin databaseAdmin, name string) error {
	if !databaseNameRE.MatchString(name) {
		return fmt.Errorf("invalid PostgreSQL database name %q; only letters, numbers and underscore are supported", name)
	}
	exists, err := admin.databaseExists(ctx, name)
	if err != nil {
		return fmt.Errorf("check database %s: %w", name, err)
	}
	if exists {
		return nil
	}
	if err := admin.createDatabase(ctx, name); err == nil {
		return nil
	} else {
		createErr := err
		exists, checkErr := admin.databaseExists(ctx, name)
		if checkErr == nil && exists {
			return nil
		}
		return fmt.Errorf("create database %s: %w", name, createErr)
	}
}

func createDatabaseSQL(name string) string {
	return "CREATE DATABASE " + pgx.Identifier{name}.Sanitize()
}

func ensureDatabase(ctx context.Context, connConfig *pgx.ConnConfig) error {
	if connConfig == nil {
		return fmt.Errorf("postgres connection config is required")
	}
	targetDatabase := strings.TrimSpace(connConfig.Database)
	if targetDatabase == "" {
		return fmt.Errorf("postgres dsn must include database name when autoCreate is enabled")
	}
	if !databaseNameRE.MatchString(targetDatabase) {
		return fmt.Errorf("invalid PostgreSQL database name %q; only letters, numbers and underscore are supported", targetDatabase)
	}
	maintenanceConfig := connConfig.Copy()
	maintenanceConfig.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, maintenanceConfig)
	if err != nil {
		return fmt.Errorf("connect postgres server for database bootstrap: %w", err)
	}
	defer conn.Close(ctx)
	return bootstrapDatabase(ctx, pgxDatabaseAdmin{conn: conn}, targetDatabase)
}
