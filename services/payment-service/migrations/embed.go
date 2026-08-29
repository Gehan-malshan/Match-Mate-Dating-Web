package migrations

import "embed"

// Files contains Payment-owned SQL migrations.
//
//go:embed *.sql
var Files embed.FS
