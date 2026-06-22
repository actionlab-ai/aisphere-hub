# PostgreSQL startup auto-create database design

## Goal

When `database.provider` is PostgreSQL and `database.autoCreate` is `true`,
startup must create the target database when it does not already exist, then
apply the existing PostgreSQL schema migration.

## Scope and configuration contract

- The target database name is read from `database.dsn`.
- `database.autoCreate: true` enables both automatic database creation and
  schema creation.
- `database.autoCreate: false` preserves strict behavior: startup fails if the
  target database is absent.
- This behavior applies only to `postgres` / `pg`; local storage is unchanged.
- The PostgreSQL role in the DSN must have permission to create databases.

## Design

`postgresstore.New` will accept an auto-create option. Before opening the
normal application pool, it will:

1. Parse the configured DSN and retain its connection parameters.
2. Connect to a maintenance database (`postgres`) using those same parameters.
3. Check `pg_database` for the configured target database.
4. Create the target database if it is absent.
5. Connect to the target database through the original DSN, ping it, and run
   the existing `AutoMigrate` table/index migration.

The database name is never interpolated directly from the DSN. It is validated
as a PostgreSQL identifier and emitted with PostgreSQL identifier quoting.

## Concurrency and failure behavior

Database creation is idempotent. If another startup creates the same database
between the existence check and `CREATE DATABASE`, startup rechecks the
catalogue and proceeds if the database now exists. Other errors retain context,
including whether the failure occurred while opening the maintenance database,
creating the target database, or migrating schema.

The application does not attempt privilege escalation. A role that lacks
`CREATEDB` receives a clear startup error rather than a partially initialized
store.

## Testing

Tests will isolate the database-bootstrap flow behind a small connector/
executor seam so they can verify without a live PostgreSQL instance:

- Existing target database: no create statement, normal target connection and
  migration.
- Absent target database: exactly one quoted `CREATE DATABASE`, then normal
  connection and migration.
- Concurrent creation: duplicate/create-conflict result followed by successful
  catalogue recheck.
- `autoCreate: false`: no maintenance connection or create attempt; missing
  database remains a startup error.
- Invalid or missing DSN database name: returns a specific error before any
  create statement.

Integration tests continue to exercise the real PostgreSQL store when
`HUB_POSTGRES_DSN` is configured.
