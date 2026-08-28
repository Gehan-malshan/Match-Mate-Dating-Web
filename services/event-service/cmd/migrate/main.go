package main

import (
	"context"
	"fmt"
	"github.com/gehan-malshan/matchmate/event-service/internal/config"
	"github.com/gehan-malshan/matchmate/event-service/migrations"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	c, err := config.Load()
	if err != nil {
		panic(err)
	}
	conn, err := pgx.Connect(ctx, c.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(614724145002)`); err != nil {
		panic(err)
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version BIGINT PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL)`); err != nil {
		panic(err)
	}
	var applied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=1)`).Scan(&applied); err != nil {
		panic(err)
	}
	if !applied {
		if _, err = tx.Exec(ctx, migrations.Up); err != nil {
			panic(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migration VALUES(1,now())`); err != nil {
			panic(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Println("event-service migration 1 ready")
}
