package main

import (
	"context"
	"fmt"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/config"
	"github.com/gehan-malshan/matchmate/matchmaking-service/migrations"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	db, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)
	tx, err := db.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(614724145003)`); err != nil {
		panic(err)
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version BIGINT PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL)`); err != nil {
		panic(err)
	}
	latest := int64(0)
	for _, migration := range migrations.All {
		var applied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=$1)`, migration.Version).Scan(&applied); err != nil {
			panic(err)
		}
		if applied {
			if migration.Version > latest {
				latest = migration.Version
			}
			continue
		}
		if _, err = tx.Exec(ctx, migration.SQL); err != nil {
			panic(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migration VALUES($1,now())`, migration.Version); err != nil {
			panic(err)
		}
		latest = migration.Version
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("matchmaking-service migrations ready through version %d\n", latest)
}
