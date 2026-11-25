package migrations

import _ "embed"

//go:embed 000001_init.up.sql
var up1 string

//go:embed 000002_consent_history.up.sql
var up2 string

type Migration struct {
	Version int64
	SQL     string
}

var All = []Migration{
	{Version: 1, SQL: up1},
	{Version: 2, SQL: up2},
}
