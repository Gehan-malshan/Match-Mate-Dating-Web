package main

import (
	"context"
	"fmt"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/migrations"
	"github.com/jackc/pgx/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
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
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(614724145007);CREATE TABLE IF NOT EXISTS schema_migration(version bigint PRIMARY KEY,applied_at timestamptz NOT NULL)`); err != nil {
		panic(err)
	}
	var applied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=1)`).Scan(&applied); err != nil {
		panic(err)
	}
	if !applied {
		sql, readErr := migrations.Files.ReadFile("000001_init.up.sql")
		if readErr != nil {
			panic(readErr)
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			panic(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migration VALUES(1,now())`); err != nil {
			panic(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Println("moderation-service migration 1 ready")
}
