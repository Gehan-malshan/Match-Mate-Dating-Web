package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/config"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

const fixtureEventID = "11111111-1111-4111-8111-000000000001"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	environment := strings.ToLower(strings.TrimSpace(cfg.Environment))
	if environment != "development" && environment != "test" {
		panic(errors.New("seed-dev is allowed only in development or test"))
	}
	ctx := context.Background()
	db, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)
	tx, err := db.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	rules := domain.Ruleset{Version: "prototype-v1", MinimumScore: 45, AllowRepeatPairings: false, MissingDataPolicy: "IGNORE_AND_RENORMALIZE", Weights: map[string]int{"relationship": 25, "personality": 20, "interests": 20, "lifestyle": 15, "values": 10, "language_location": 10}}
	rawRules, _ := json.Marshal(rules)
	if _, err = tx.Exec(ctx, `INSERT INTO ruleset(version,status,configuration,approved_by,approved_at) VALUES($1,'APPROVED',$2,'development-product-owner',$3) ON CONFLICT(version) DO NOTHING`, rules.Version, rawRules, now); err != nil {
		panic(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO event_scope(event_id,organizer_id,event_status,ruleset_version,source_version,updated_at) VALUES($1,$2,'MATCHING',$3,1,$4) ON CONFLICT(event_id) DO UPDATE SET organizer_id=EXCLUDED.organizer_id,event_status=EXCLUDED.event_status,ruleset_version=EXCLUDED.ruleset_version,source_version=event_scope.source_version+1,updated_at=EXCLUDED.updated_at`, fixtureEventID, "00000000-0000-4000-8000-000000000005", rules.Version, now); err != nil {
		panic(err)
	}
	for index, p := range fixtures() {
		raw, _ := json.Marshal(p)
		if _, err = tx.Exec(ctx, `INSERT INTO participant_projection(event_id,account_id,participant_code,group_code,input,source_version,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(event_id,account_id) DO UPDATE SET participant_code=EXCLUDED.participant_code,group_code=EXCLUDED.group_code,input=EXCLUDED.input,source_version=EXCLUDED.source_version,updated_at=EXCLUDED.updated_at`, fixtureEventID, p.AccountID, p.ParticipantCode, p.Group, raw, index+1, now); err != nil {
			panic(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("Seeded ruleset %s and %d participants for event %s\n", rules.Version, len(fixtures()), fixtureEventID)
}

func fixtures() []domain.Participant {
	return []domain.Participant{
		participant("00000000-0000-4000-8000-000000000001", "MM-A01", "A", 29, "long-term", []string{"books", "travel", "live music"}, []string{"kindness", "growth"}, []string{"English", "Sinhala"}, "Colombo 05", 3, "active", "no"),
		participant("10000000-0000-4000-8000-000000000001", "MM-A02", "A", 29, "long-term", []string{"architecture", "coffee", "weekend walks"}, []string{"curiosity", "growth"}, []string{"English"}, "Colombo 05", 2, "balanced", "no"),
		participant("10000000-0000-4000-8000-000000000003", "MM-A03", "A", 28, "intentional", []string{"books", "art", "cooking"}, []string{"kindness", "family"}, []string{"English", "Sinhala"}, "Nugegoda", 3, "active", "no"),
		participant("10000000-0000-4000-8000-000000000005", "MM-A04", "A", 33, "long-term", []string{"travel", "design", "tennis"}, []string{"ambition", "growth"}, []string{"English"}, "Colombo 03", 4, "active", "no"),
		participant("20000000-0000-4000-8000-000000000001", "MM-A05", "A", 31, "long-term", []string{"fitness"}, []string{"growth"}, []string{"English"}, "Colombo", 3, "active", "no"),
		participant("00000000-0000-4000-8000-000000000002", "MM-B01", "B", 31, "long-term", []string{"books", "music", "travel"}, []string{"kindness", "growth"}, []string{"English", "Sinhala"}, "Colombo 05", 3, "active", "no"),
		participant("10000000-0000-4000-8000-000000000002", "MM-B02", "B", 31, "long-term", []string{"live music", "photography", "sri lankan food"}, []string{"curiosity", "growth"}, []string{"English"}, "Colombo 07", 2, "balanced", "no"),
		participant("10000000-0000-4000-8000-000000000004", "MM-B03", "B", 30, "intentional", []string{"film", "hiking", "board games"}, []string{"kindness", "family"}, []string{"English", "Sinhala"}, "Rajagiriya", 3, "active", "no"),
		participant("10000000-0000-4000-8000-000000000006", "MM-B04", "B", 29, "long-term", []string{"podcasts", "beach walks", "yoga"}, []string{"ambition", "growth"}, []string{"English"}, "Mount Lavinia", 4, "active", "no"),
		func() domain.Participant {
			p := participant("20000000-0000-4000-8000-000000000002", "MM-B05", "B", 30, "long-term", []string{"fitness"}, []string{"growth"}, []string{"English"}, "Colombo", 3, "active", "yes")
			p.BookingConfirmed = false
			return p
		}(),
	}
}

func participant(id, code, group string, age int, intention string, interests, values, languages []string, location string, social int, activity, smoking string) domain.Participant {
	accepted := "A"
	if group == "A" {
		accepted = "B"
	}
	return domain.Participant{AccountID: id, ParticipantCode: code, Group: group, AcceptedGroups: []string{accepted}, Age: age, MinimumAge: 25, MaximumAge: 38, Active: true, Verified: true, ProfileApproved: true, BookingConfirmed: true, RelationshipIntent: intention, Interests: interests, Personality: map[string]int{"social": social, "planning": 3}, Lifestyle: map[string]string{"activity": activity, "smoking": smoking}, Values: values, Languages: languages, BroadLocation: location, DealBreakers: map[string]string{"smoking": "yes"}}
}
