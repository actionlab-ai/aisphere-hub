package main

import (
	"testing"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/config"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
)

func TestInitStorePassesPostgresAutoCreateSetting(t *testing.T) {
	original := newPostgresStore
	t.Cleanup(func() { newPostgresStore = original })

	var gotDSN string
	var gotAutoCreate bool
	newPostgresStore = func(dsn string, autoCreate bool) (store.Backend, error) {
		gotDSN, gotAutoCreate = dsn, autoCreate
		return nil, nil
	}

	var cfg config.Config
	cfg.Database.Provider = "postgres"
	cfg.Database.DSN = "host=example dbname=aisphere_hub"
	cfg.Database.AutoCreate = true
	if _, err := initStore(cfg); err != nil {
		t.Fatal(err)
	}
	if gotDSN != cfg.Database.DSN || !gotAutoCreate {
		t.Fatalf("factory called with dsn=%q autoCreate=%v", gotDSN, gotAutoCreate)
	}
}
