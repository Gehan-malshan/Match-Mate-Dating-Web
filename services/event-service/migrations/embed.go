package migrations

import _ "embed"

//go:embed 000001_init.up.sql
var Up string

//go:embed 000002_payment_options.up.sql
var PaymentOptionsUp string

//go:embed 000003_event_image.up.sql
var EventImageUp string
