package postgresstore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeDatabaseAdmin struct {
	exists          bool
	existsAfterFail bool
	createErr       error
	created         []string
	checks          int
}

func (a *fakeDatabaseAdmin) databaseExists(_ context.Context, _ string) (bool, error) {
	a.checks++
	if a.checks > 1 {
		return a.existsAfterFail, nil
	}
	return a.exists, nil
}

func (a *fakeDatabaseAdmin) createDatabase(_ context.Context, name string) error {
	a.created = append(a.created, name)
	return a.createErr
}

func TestBootstrapDatabaseCreatesMissingDatabase(t *testing.T) {
	admin := &fakeDatabaseAdmin{}
	if err := bootstrapDatabase(context.Background(), admin, "aisphere_hub"); err != nil {
		t.Fatal(err)
	}
	if got, want := admin.created, []string{"aisphere_hub"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("created = %v, want %v", got, want)
	}
}

func TestBootstrapDatabaseSkipsExistingDatabase(t *testing.T) {
	admin := &fakeDatabaseAdmin{exists: true}
	if err := bootstrapDatabase(context.Background(), admin, "aisphere_hub"); err != nil {
		t.Fatal(err)
	}
	if len(admin.created) != 0 {
		t.Fatalf("unexpected create: %v", admin.created)
	}
}

func TestBootstrapDatabaseAcceptsConcurrentCreate(t *testing.T) {
	admin := &fakeDatabaseAdmin{createErr: errors.New("duplicate database"), existsAfterFail: true}
	if err := bootstrapDatabase(context.Background(), admin, "aisphere_hub"); err != nil {
		t.Fatal(err)
	}
	if got, want := admin.checks, 2; got != want {
		t.Fatalf("checks = %d, want %d", got, want)
	}
}

func TestBootstrapDatabaseRejectsInvalidDatabaseName(t *testing.T) {
	err := bootstrapDatabase(context.Background(), &fakeDatabaseAdmin{}, "bad-name")
	if err == nil || !strings.Contains(err.Error(), "invalid PostgreSQL database name") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateDatabaseSQLQuotesIdentifier(t *testing.T) {
	if got, want := createDatabaseSQL("aisphere_hub"), `CREATE DATABASE "aisphere_hub"`; got != want {
		t.Fatalf("SQL = %q, want %q", got, want)
	}
}
