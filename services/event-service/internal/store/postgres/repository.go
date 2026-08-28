package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/gehan-malshan/matchmate/event-service/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repository struct{ pool *pgxpool.Pool }

func New(p *pgxpool.Pool) *Repository                { return &Repository{pool: p} }
func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }
func (r *Repository) Create(ctx context.Context, e domain.Event, f domain.Fact) (domain.Event, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return e, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO event(event_id,organizer_id,name,description,venue_name,broad_location,venue_time_zone,starts_at,ends_at,registration_opens_at,registration_closes_at,price,currency,configured_capacity,capacity_policy_version,matching_ruleset_version,status,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::numeric,$13,$14,$15,$16,$17,$18,$19,$20)`, e.ID, e.OrganizerID, e.Name, e.Description, e.VenueName, e.BroadLocation, e.TimeZone, e.StartsAt, e.EndsAt, e.RegistrationOpensAt, e.RegistrationClosesAt, e.Price, e.Currency, e.ConfiguredCapacity, e.CapacityPolicyVersion, e.MatchingRulesetVersion, e.Status, e.Version, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return e, err
	}
	if err = insertFact(ctx, tx, f, e); err != nil {
		return e, err
	}
	return e, tx.Commit(ctx)
}

const selectEvent = `SELECT event_id::text,organizer_id,name,description,venue_name,broad_location,venue_time_zone,starts_at,ends_at,registration_opens_at,registration_closes_at,price::text,currency,configured_capacity,capacity_policy_version,matching_ruleset_version,status,version,created_at,updated_at FROM event`

type row interface{ Scan(...any) error }

func scan(q row) (domain.Event, error) {
	var e domain.Event
	err := q.Scan(&e.ID, &e.OrganizerID, &e.Name, &e.Description, &e.VenueName, &e.BroadLocation, &e.TimeZone, &e.StartsAt, &e.EndsAt, &e.RegistrationOpensAt, &e.RegistrationClosesAt, &e.Price, &e.Currency, &e.ConfiguredCapacity, &e.CapacityPolicyVersion, &e.MatchingRulesetVersion, &e.Status, &e.Version, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}
func (r *Repository) Get(ctx context.Context, id string) (domain.Event, error) {
	e, err := scan(r.pool.QueryRow(ctx, selectEvent+` WHERE event_id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		err = store.ErrNotFound
	}
	return e, err
}
func (r *Repository) Update(ctx context.Context, id string, in domain.UpdateInput, f domain.Fact) (domain.Event, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Event{}, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	e, err := scan(tx.QueryRow(ctx, `UPDATE event SET organizer_id=$3,name=$4,description=$5,venue_name=$6,broad_location=$7,venue_time_zone=$8,starts_at=$9,ends_at=$10,registration_opens_at=$11,registration_closes_at=$12,price=$13::numeric,currency=$14,configured_capacity=$15,capacity_policy_version=CASE WHEN configured_capacity<>$15 THEN capacity_policy_version+1 ELSE capacity_policy_version END,matching_ruleset_version=$16,version=version+1,updated_at=$17 WHERE event_id=$1 AND version=$2 RETURNING event_id::text,organizer_id,name,description,venue_name,broad_location,venue_time_zone,starts_at,ends_at,registration_opens_at,registration_closes_at,price::text,currency,configured_capacity,capacity_policy_version,matching_ruleset_version,status,version,created_at,updated_at`, id, in.ExpectedVersion, in.OrganizerID, in.Name, in.Description, in.VenueName, in.BroadLocation, in.TimeZone, in.StartsAt.UTC(), in.EndsAt.UTC(), in.RegistrationOpensAt.UTC(), in.RegistrationClosesAt.UTC(), in.Price, in.Currency, in.ConfiguredCapacity, in.MatchingRulesetVersion, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return e, store.ErrConflict
	}
	if err != nil {
		return e, err
	}
	f.AggregateID = e.ID
	f.Payload = map[string]any{"eventId": e.ID, "version": e.Version}
	if err = insertFact(ctx, tx, f, e); err != nil {
		return e, err
	}
	return e, tx.Commit(ctx)
}
func (r *Repository) Transition(ctx context.Context, id string, expected int64, to domain.Status, reason string, f domain.Fact) (domain.Event, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Event{}, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	e, err := scan(tx.QueryRow(ctx, `UPDATE event SET status=$3,version=version+1,updated_at=$4 WHERE event_id=$1 AND version=$2 RETURNING event_id::text,organizer_id,name,description,venue_name,broad_location,venue_time_zone,starts_at,ends_at,registration_opens_at,registration_closes_at,price::text,currency,configured_capacity,capacity_policy_version,matching_ruleset_version,status,version,created_at,updated_at`, id, expected, to, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return e, store.ErrConflict
	}
	if err != nil {
		return e, err
	}
	before, _ := json.Marshal(map[string]any{"status": f.Payload["status"]})
	after, _ := json.Marshal(map[string]any{"status": e.Status})
	_, err = tx.Exec(ctx, `INSERT INTO event_audit(audit_id,event_id,actor_id,action,reason,prior_state,new_state,correlation_id,occurred_at)VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8)`, id, f.ActorID, f.EventType, reason, before, after, f.CorrelationID, now)
	if err != nil {
		return e, err
	}
	f.AggregateID = e.ID
	f.Payload = map[string]any{"eventId": e.ID, "status": e.Status, "version": e.Version}
	if err = insertFact(ctx, tx, f, e); err != nil {
		return e, err
	}
	return e, tx.Commit(ctx)
}
func (r *Repository) ListDiscoverable(ctx context.Context, cursor string, limit int, now time.Time) (domain.Page, error) {
	cursorTime := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	cursorID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	if cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return domain.Page{}, &domain.ProblemError{Status: 422, Code: "EVENT_CURSOR_INVALID", Detail: "Cursor is invalid"}
		}
		var c struct {
			T  time.Time `json:"t"`
			ID string    `json:"id"`
		}
		if json.Unmarshal(b, &c) != nil || c.ID == "" {
			return domain.Page{}, &domain.ProblemError{Status: 422, Code: "EVENT_CURSOR_INVALID", Detail: "Cursor is invalid"}
		}
		cursorTime, cursorID = c.T, c.ID
	}
	rows, err := r.pool.Query(ctx, selectEvent+` WHERE status IN ('PUBLISHED','REGISTRATION_OPEN','REGISTRATION_CLOSED') AND starts_at>$1 AND (starts_at,event_id)<($2,$3::uuid) ORDER BY starts_at DESC,event_id DESC LIMIT $4`, now, cursorTime, cursorID, limit+1)
	if err != nil {
		return domain.Page{}, err
	}
	defer rows.Close()
	p := domain.Page{Items: []domain.Event{}, Limit: limit}
	for rows.Next() {
		e, er := scan(rows)
		if er != nil {
			return p, er
		}
		p.Items = append(p.Items, e)
	}
	if len(p.Items) > limit {
		last := p.Items[limit-1]
		raw, _ := json.Marshal(map[string]any{"t": last.StartsAt, "id": last.ID})
		p.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
		p.Items = p.Items[:limit]
	}
	return p, rows.Err()
}
func (r *Repository) ListManaged(ctx context.Context, organizer string, isAdmin bool, cursor string, limit int) (domain.Page, error) {
	cursorTime := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	cursorID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	if cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return domain.Page{}, &domain.ProblemError{Status: 422, Code: "EVENT_CURSOR_INVALID", Detail: "Cursor is invalid"}
		}
		var c struct {
			T  time.Time `json:"t"`
			ID string    `json:"id"`
		}
		if json.Unmarshal(b, &c) != nil || c.ID == "" {
			return domain.Page{}, &domain.ProblemError{Status: 422, Code: "EVENT_CURSOR_INVALID", Detail: "Cursor is invalid"}
		}
		cursorTime, cursorID = c.T, c.ID
	}
	rows, err := r.pool.Query(ctx, selectEvent+` WHERE ($1 OR organizer_id=$2) AND (updated_at,event_id)<($3,$4::uuid) ORDER BY updated_at DESC,event_id DESC LIMIT $5`, isAdmin, organizer, cursorTime, cursorID, limit+1)
	if err != nil {
		return domain.Page{}, err
	}
	defer rows.Close()
	p := domain.Page{Items: []domain.Event{}, Limit: limit}
	for rows.Next() {
		e, er := scan(rows)
		if er != nil {
			return p, er
		}
		p.Items = append(p.Items, e)
	}
	if len(p.Items) > limit {
		last := p.Items[limit-1]
		raw, _ := json.Marshal(map[string]any{"t": last.UpdatedAt, "id": last.ID})
		p.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
		p.Items = p.Items[:limit]
	}
	return p, rows.Err()
}
func insertFact(ctx context.Context, tx pgx.Tx, f domain.Fact, e domain.Event) error {
	payload, _ := json.Marshal(f.Payload)
	_, err := tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,occurred_at,aggregate_id,correlation_id,actor_id,routing_key,payload)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, f.EventID, f.EventType, f.SchemaVersion, f.OccurredAt, e.ID, f.CorrelationID, f.ActorID, "event."+f.EventType, payload)
	return err
}
func (r *Repository) ClaimOutbox(ctx context.Context, limit int) ([]store.OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH picked AS (SELECT event_id FROM outbox WHERE published_at IS NULL AND (claimed_at IS NULL OR claimed_at<now()-interval '1 minute') ORDER BY occurred_at LIMIT $1 FOR UPDATE SKIP LOCKED) UPDATE outbox o SET claimed_at=now(),attempts=attempts+1 FROM picked WHERE o.event_id=picked.event_id RETURNING o.event_id::text,o.routing_key,jsonb_build_object('eventId',o.event_id,'eventType',o.event_type,'schemaVersion',o.schema_version,'occurredAt',o.occurred_at,'aggregateId',o.aggregate_id,'correlationId',o.correlation_id,'causationId',o.causation_id,'actorId',o.actor_id,'payload',o.payload)::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.OutboxRecord{}
	for rows.Next() {
		var v store.OutboxRecord
		var body string
		if err = rows.Scan(&v.ID, &v.RoutingKey, &body); err != nil {
			return nil, err
		}
		v.Body = []byte(body)
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at=$2,claimed_at=NULL WHERE event_id=$1`, id, at)
	return err
}
