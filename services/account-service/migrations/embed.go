package migrations

import _ "embed"

//go:embed 000001_init.up.sql
var Up string

//go:embed 000002_security.up.sql
var SecurityUp string
