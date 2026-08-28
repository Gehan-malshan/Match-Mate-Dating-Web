// Command seed-dev installs deterministic development fixtures. It is intentionally
// guarded to APP_ENV=development and is started by Docker Compose after migrations.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gehan-malshan/matchmate/event-service/internal/config"
	"github.com/jackc/pgx/v5"
)

type fixture struct {
	id, name, description, location, venue, status, price string
	dayOffset                                             int
	hour                                                  int
	capacity                                              int
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Environment != "development" {
		log.Fatal("development event fixtures may only run with APP_ENV=development")
	}
	db, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(ctx)

	colombo, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		log.Fatal(err)
	}
	today := time.Now().In(colombo)
	base := time.Date(today.Year(), today.Month(), today.Day(), 19, 0, 0, 0, colombo).AddDate(0, 0, 7)
	fixtures := []fixture{
		{"11111111-1111-4111-8111-000000000001", "Midnight Rooftop Social", "A hosted evening of guided small-group conversations and calm, respectful introductions.", "Colombo 03", "Development Rooftop", "REGISTRATION_OPEN", "3500.00", 0, 0, 48},
		{"11111111-1111-4111-8111-000000000002", "Café Conversation Circle", "A daytime gathering for thoughtful conversation in a comfortable, moderator-supported setting.", "Colombo 07", "Development Café", "REGISTRATION_OPEN", "2500.00", 7, -1, 32},
		{"11111111-1111-4111-8111-000000000003", "Gallery After Dark", "An evening of art, prompts, and intentional introductions for members ready to meet in person.", "Colombo 01", "Development Gallery", "PUBLISHED", "4000.00", 14, 0, 40},
		{"11111111-1111-4111-8111-000000000004", "Seaside Sunset Circle", "A relaxed sunset event with facilitated conversation and no pressure to share personal contact details.", "Mount Lavinia", "Development Seaside Venue", "PUBLISHED", "3000.00", 21, -1, 36},
	}
	const statement = `INSERT INTO event (event_id, organizer_id, name, description, venue_name, broad_location, venue_time_zone, starts_at, ends_at, registration_opens_at, registration_closes_at, price, currency, configured_capacity, matching_ruleset_version, status, created_at, updated_at)
VALUES ($1::uuid, 'development-organizer', $2, $3, $4, $5, 'Asia/Colombo', $6, $7, $8, $9, $10::numeric, 'LKR', $11, 'development-v1', $12, $13, $13)
ON CONFLICT (event_id) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description, venue_name=EXCLUDED.venue_name, broad_location=EXCLUDED.broad_location, starts_at=EXCLUDED.starts_at, ends_at=EXCLUDED.ends_at, registration_opens_at=EXCLUDED.registration_opens_at, registration_closes_at=EXCLUDED.registration_closes_at, price=EXCLUDED.price, configured_capacity=EXCLUDED.configured_capacity, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`
	for _, f := range fixtures {
		starts := base.AddDate(0, 0, f.dayOffset).Add(time.Duration(f.hour) * time.Hour)
		ends := starts.Add(3 * time.Hour)
		now := time.Now().UTC()
		_, err = db.Exec(ctx, statement, f.id, f.name, f.description, f.venue, f.location, starts, ends, starts.AddDate(0, 0, -14), starts.AddDate(0, 0, -1), f.price, f.capacity, f.status, now)
		if err != nil {
			log.Fatalf("seed %s: %v", f.name, err)
		}
	}
	fmt.Println("Seeded 4 shared development events")
}
