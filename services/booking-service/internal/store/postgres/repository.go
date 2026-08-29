package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repository struct{ pool *pgxpool.Pool }
type OutboxRecord struct {
	ID, RoutingKey string
	Body           []byte
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const cols = `id,account_id,event_id,state,amount::text,currency,policy_version,expires_at,version,created_at,confirmed_at,cancelled_at`

func scan(row pgx.Row) (domain.Booking, error) {
	var b domain.Booking
	err := row.Scan(&b.ID, &b.AccountID, &b.EventID, &b.State, &b.Amount, &b.Currency, &b.PolicyVersion, &b.ExpiresAt, &b.Version, &b.CreatedAt, &b.ConfirmedAt, &b.CancelledAt)
	return b, err
}
func (r *Repository) Ready(ctx context.Context) error { return r.pool.Ping(ctx) }
func (r *Repository) Create(ctx context.Context, b domain.Booking, key, fingerprint string, capacity int) (domain.Booking, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return b, false, err
	}
	defer tx.Rollback(ctx)
	var fp, id string
	err = tx.QueryRow(ctx, `SELECT request_fingerprint,booking_id FROM idempotency_record WHERE actor_id=$1 AND operation='booking.create' AND idempotency_key=$2`, b.AccountID, key).Scan(&fp, &id)
	if err == nil {
		if fp != fingerprint {
			return b, false, application.ErrConflict
		}
		existing, e := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1`, id))
		return existing, true, e
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return b, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO capacity_allocation(event_id,configured_capacity,policy_version) VALUES($1,$2,$3) ON CONFLICT(event_id) DO NOTHING`, b.EventID, capacity, b.PolicyVersion)
	if err != nil {
		return b, false, err
	}
	tag, err := tx.Exec(ctx, `UPDATE capacity_allocation SET held_count=held_count+1,version=version+1 WHERE event_id=$1 AND policy_version=$2 AND held_count+confirmed_count<configured_capacity`, b.EventID, b.PolicyVersion)
	if err != nil {
		return b, false, err
	}
	if tag.RowsAffected() != 1 {
		return b, false, application.ErrCapacity
	}
	_, err = tx.Exec(ctx, `INSERT INTO booking(id,account_id,event_id,state,amount,currency,policy_version,expires_at,version,created_at) VALUES($1,$2,$3,'PENDING_PAYMENT',$4,$5,$6,$7,1,$8)`, b.ID, b.AccountID, b.EventID, b.Amount, b.Currency, b.PolicyVersion, b.ExpiresAt, b.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return b, false, application.ErrConflict
		}
		return b, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_record(actor_id,operation,idempotency_key,request_fingerprint,booking_id,created_at,expires_at) VALUES($1,'booking.create',$2,$3,$4,$5,$5::timestamptz+interval '24 hours')`, b.AccountID, key, fingerprint, b.ID, b.CreatedAt)
	if err != nil {
		return b, false, err
	}
	if err = insertFact(ctx, tx, "BookingPending", b, "", b.AccountID, b.CreatedAt); err != nil {
		return b, false, err
	}
	return b, false, tx.Commit(ctx)
}
func (r *Repository) Get(ctx context.Context, actor, id string) (domain.Booking, error) {
	b, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1 AND account_id=$2`, id, actor))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, application.ErrNotFound
	}
	return b, err
}
func (r *Repository) List(ctx context.Context, actor string, limit int) ([]domain.Booking, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM booking WHERE account_id=$1 ORDER BY created_at DESC LIMIT $2`, actor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Booking, 0)
	for rows.Next() {
		b, e := scan(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, b)
	}
	return items, rows.Err()
}
func (r *Repository) Cancel(ctx context.Context, actor, id, key string, now time.Time) (domain.Booking, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Booking{}, false, err
	}
	defer tx.Rollback(ctx)
	var fp, existingID string
	err = tx.QueryRow(ctx, `SELECT request_fingerprint,booking_id FROM idempotency_record WHERE actor_id=$1 AND operation='booking.cancel' AND idempotency_key=$2`, actor, key).Scan(&fp, &existingID)
	if err == nil {
		if fp != id {
			return domain.Booking{}, false, application.ErrConflict
		}
		b, e := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1 AND account_id=$2`, existingID, actor))
		return b, true, e
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, false, err
	}
	b, err := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1 AND account_id=$2 FOR UPDATE`, id, actor))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, false, application.ErrNotFound
	}
	if err != nil {
		return b, false, err
	}
	if b.State == domain.Cancelled {
		return b, true, tx.Commit(ctx)
	}
	if b.State != domain.Pending {
		return b, false, application.ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE booking SET state='CANCELLED',cancelled_at=$2,version=version+1 WHERE id=$1`, b.ID, now)
	if err != nil {
		return b, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE capacity_allocation SET held_count=held_count-1,version=version+1 WHERE event_id=$1 AND held_count>0`, b.EventID)
	if err != nil {
		return b, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_record(actor_id,operation,idempotency_key,request_fingerprint,booking_id,created_at,expires_at) VALUES($1,'booking.cancel',$2,$3,$4,$5,$5::timestamptz+interval '24 hours')`, actor, key, id, b.ID, now)
	if err != nil {
		return b, false, err
	}
	b.State = domain.Cancelled
	b.CancelledAt = &now
	b.Version++
	if err = insertFact(ctx, tx, "BookingCancelled", b, "", actor, now); err != nil {
		return b, false, err
	}
	return b, false, tx.Commit(ctx)
}
func (r *Repository) Expire(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT `+cols+` FROM booking WHERE state='PENDING_PAYMENT' AND expires_at<=$1 ORDER BY expires_at LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return 0, err
	}
	var list []domain.Booking
	for rows.Next() {
		b, e := scan(rows)
		if e != nil {
			rows.Close()
			return 0, e
		}
		list = append(list, b)
	}
	rows.Close()
	for _, b := range list {
		_, err = tx.Exec(ctx, `UPDATE booking SET state='EXPIRED',version=version+1 WHERE id=$1`, b.ID)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, `UPDATE capacity_allocation SET held_count=held_count-1,version=version+1 WHERE event_id=$1 AND held_count>0`, b.EventID)
		if err != nil {
			return 0, err
		}
		b.State = domain.Expired
		if err = insertFact(ctx, tx, "HoldExpired", b, "", "system:booking-expiry", now); err != nil {
			return 0, err
		}
	}
	return len(list), tx.Commit(ctx)
}
func insertFact(ctx context.Context, tx pgx.Tx, eventType string, b domain.Booking, causation, actor string, at time.Time) error {
	payload, _ := json.Marshal(map[string]any{"bookingId": b.ID, "eventId": b.EventID, "accountId": b.AccountID, "amount": b.Amount, "currency": b.Currency, "state": b.State})
	_, err := tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,aggregate_id,correlation_id,causation_id,actor_id,payload,occurred_at) VALUES($1,$2,1,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8)`, uuid.NewString(), eventType, b.ID, uuid.NewString(), causation, actor, payload, at)
	return err
}
func (r *Repository) ClaimOutbox(ctx context.Context, limit int) ([]OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH picked AS (SELECT event_id FROM outbox WHERE published_at IS NULL AND(claimed_at IS NULL OR claimed_at<now()-interval '1 minute') ORDER BY occurred_at LIMIT $1 FOR UPDATE SKIP LOCKED) UPDATE outbox o SET claimed_at=now(),attempts=attempts+1 FROM picked WHERE o.event_id=picked.event_id RETURNING o.event_id::text,'booking.'||o.event_type,jsonb_build_object('eventId',o.event_id,'eventType',o.event_type,'schemaVersion',o.schema_version,'occurredAt',o.occurred_at,'aggregateId',o.aggregate_id,'correlationId',o.correlation_id,'causationId',o.causation_id,'actorId',o.actor_id,'payload',o.payload)::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxRecord
	for rows.Next() {
		var x OutboxRecord
		var body string
		if err = rows.Scan(&x.ID, &x.RoutingKey, &body); err != nil {
			return nil, err
		}
		x.Body = []byte(body)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at=$2,claimed_at=NULL WHERE event_id=$1`, id, at)
	return err
}

func (r *Repository) ApplyPaymentEvent(ctx context.Context, eventID, eventType, paymentID, bookingID string, at time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `INSERT INTO inbox(consumer,event_id,processed_at) VALUES('booking.payment.v1',$1,$2) ON CONFLICT DO NOTHING`, eventID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	b, err := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1 FOR UPDATE`, bookingID))
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("payment references unknown booking")
	}
	if err != nil {
		return err
	}
	if eventType == "PaymentCompleted" {
		if b.State == domain.Pending && b.ExpiresAt.After(at) {
			_, err = tx.Exec(ctx, `UPDATE booking SET state='CONFIRMED',confirmed_at=$2,version=version+1 WHERE id=$1`, b.ID, at)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE capacity_allocation SET held_count=held_count-1,confirmed_count=confirmed_count+1,version=version+1 WHERE event_id=$1 AND held_count>0`, b.EventID)
			if err != nil {
				return err
			}
			b.State = domain.Confirmed
			b.ConfirmedAt = &at
			if err = insertFact(ctx, tx, "BookingConfirmed", b, eventID, "system:payment", at); err != nil {
				return err
			}
		} else if b.State == domain.Pending || b.State == domain.Expired {
			if b.State == domain.Pending {
				_, err = tx.Exec(ctx, `UPDATE capacity_allocation SET held_count=held_count-1,version=version+1 WHERE event_id=$1 AND held_count>0`, b.EventID)
				if err != nil {
					return err
				}
			}
			_, err = tx.Exec(ctx, `UPDATE booking SET state='PAYMENT_REVIEW',version=version+1 WHERE id=$1`, b.ID)
			if err != nil {
				return err
			}
			b.State = domain.PaymentReview
			if err = insertFact(ctx, tx, "BookingPaymentReviewRequired", b, eventID, "system:payment", at); err != nil {
				return err
			}
		}
	}
	_ = paymentID
	return tx.Commit(ctx)
}
