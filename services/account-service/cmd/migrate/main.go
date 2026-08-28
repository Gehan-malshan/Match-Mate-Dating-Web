package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/gehan-malshan/matchmate/account-service/internal/config"
	"github.com/gehan-malshan/matchmate/account-service/migrations"
	"github.com/jackc/pgx/v5"
)

const migrationVersion int64 = 1

var versionOneTables = []string{
	"account", "credential", "consent_record", "role_assignment",
	"email_verification_token", "refresh_session", "profile", "profile_interest",
	"matching_preference", "member_block", "audit_log", "outbox",
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(614724145001)`); err != nil {
		panic(err)
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version BIGINT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL)`); err != nil {
		panic(err)
	}
	var applied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=$1)`, migrationVersion).Scan(&applied); err != nil {
		panic(err)
	}
	if applied {
		if err = tx.Commit(ctx); err != nil {
			panic(err)
		}
		fmt.Printf("account-service migration %d already applied\n", migrationVersion)
		return
	}

	existing, err := existingVersionOneTables(ctx, tx)
	if err != nil {
		panic(err)
	}
	switch {
	case existing == 0:
		if _, err = tx.Exec(ctx, string(migrations.Up)); err != nil {
			panic(err)
		}
	case existing == len(versionOneTables):
		fmt.Println("existing account schema detected; recording migration baseline")
	default:
		panic(errors.New("partial account schema detected; restore/reset the development database before migrating"))
	}
	if _, err = tx.Exec(ctx, `INSERT INTO schema_migration(version,applied_at) VALUES($1,now())`, migrationVersion); err != nil {
		panic(err)
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("account-service migration %d applied\n", migrationVersion)
}

func existingVersionOneTables(ctx context.Context, tx pgx.Tx) (int, error) {
	existing := 0
	for _, table := range versionOneTables {
		var name *string
		if err := tx.QueryRow(ctx, `SELECT to_regclass($1)::text`, "public."+table).Scan(&name); err != nil {
			return 0, err
		}
		if name != nil {
			existing++
		}
	}
	return existing, nil
}
