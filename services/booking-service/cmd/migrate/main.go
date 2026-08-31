package main

import (
	"context"
	"fmt"
	"github.com/gehan-malshan/matchmate/booking-service/migrations"
	"github.com/jackc/pgx/v5"
	"os"
)

func main() {
	u := os.Getenv("BOOKING_DATABASE_URL")
	if u == "" {
		panic("BOOKING_DATABASE_URL required")
	}
	ctx := context.Background()
	c, e := pgx.Connect(ctx, u)
	if e != nil {
		panic(e)
	}
	defer c.Close(ctx)
	if _, e = c.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		panic(e)
	}
	for version, file := range []string{"000001_init.up.sql", "000002_cancellation.up.sql"} {
		v := version + 1
		var applied bool
		if e = c.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=$1)`, v).Scan(&applied); e != nil {
			panic(e)
		}
		if applied {
			continue
		}
		sql, readErr := migrations.Files.ReadFile(file)
		if readErr != nil {
			panic(readErr)
		}
		tx, beginErr := c.Begin(ctx)
		if beginErr != nil {
			panic(beginErr)
		}
		if _, e = tx.Exec(ctx, string(sql)); e != nil {
			_ = tx.Rollback(ctx)
			panic(fmt.Errorf("migration %d: %w", v, e))
		}
		if _, e = tx.Exec(ctx, `INSERT INTO schema_migration(version) VALUES($1)`, v); e != nil {
			_ = tx.Rollback(ctx)
			panic(e)
		}
		if e = tx.Commit(ctx); e != nil {
			panic(e)
		}
	}
}
