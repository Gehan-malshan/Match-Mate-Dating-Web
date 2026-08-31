package main

import (
	"context"
	"fmt"
	"github.com/gehan-malshan/matchmate/payment-service/migrations"
	"github.com/jackc/pgx/v5"
	"os"
)

func main() {
	url := os.Getenv("PAYMENT_DATABASE_URL")
	if url == "" {
		panic("PAYMENT_DATABASE_URL is required")
	}
	ctx := context.Background()
	c, e := pgx.Connect(ctx, url)
	if e != nil {
		panic(e)
	}
	defer c.Close(ctx)
	if _, e = c.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migration(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		panic(e)
	}
	var applied bool
	if e = c.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migration WHERE version=1)`).Scan(&applied); e != nil {
		panic(e)
	}
	if applied {
		return
	}
	sql, e := migrations.Files.ReadFile("000001_init.up.sql")
	if e != nil {
		panic(e)
	}
	tx, e := c.Begin(ctx)
	if e != nil {
		panic(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, string(sql)); e != nil {
		panic(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO schema_migration(version) VALUES(1)`); e != nil {
		panic(e)
	}
	if e = tx.Commit(ctx); e != nil {
		panic(e)
	}
	fmt.Println("payment migration 000001 applied")
}
