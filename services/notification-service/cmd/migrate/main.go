package main

import (
	"context"
	"fmt"

	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/migrations"
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
	if _, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		panic(err)
	}

	for index, filename := range []string{"000001_init.up.sql", "000002_member_feed.up.sql"} {
		version := index + 1
		var applied bool
		if err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=$1)`, version).Scan(&applied); err != nil {
			panic(err)
		}
		if applied {
			continue
		}
		sql, readErr := migrations.Files.ReadFile(filename)
		if readErr != nil {
			panic(readErr)
		}
		tx, beginErr := conn.Begin(ctx)
		if beginErr != nil {
			panic(beginErr)
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			panic(fmt.Errorf("notification migration %d: %w", version, err))
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migration(version) VALUES($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			panic(err)
		}
		if err = tx.Commit(ctx); err != nil {
			panic(err)
		}
	}
	fmt.Println("notification-service migrations ready")
}
