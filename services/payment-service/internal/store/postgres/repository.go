package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/application"
	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }
type OutboxRecord struct {
	ID, RoutingKey string
	Body           []byte
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) ClaimOutbox(ctx context.Context, limit int) ([]OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH picked AS (SELECT event_id FROM outbox WHERE published_at IS NULL AND (claimed_at IS NULL OR claimed_at < now()-interval '1 minute') ORDER BY occurred_at LIMIT $1 FOR UPDATE SKIP LOCKED) UPDATE outbox o SET claimed_at=now(),attempts=attempts+1 FROM picked WHERE o.event_id=picked.event_id RETURNING o.event_id::text,'payment.'||o.event_type,jsonb_build_object('eventId',o.event_id,'eventType',o.event_type,'schemaVersion',o.schema_version,'occurredAt',o.occurred_at,'aggregateId',o.aggregate_id,'correlationId',o.correlation_id,'causationId',o.causation_id,'actorId',o.actor_id,'payload',o.payload)::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []OutboxRecord
	for rows.Next() {
		var record OutboxRecord
		var body string
		if err = rows.Scan(&record.ID, &record.RoutingKey, &body); err != nil {
			return nil, err
		}
		record.Body = []byte(body)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at=$2,claimed_at=NULL WHERE event_id=$1`, id, at)
	return err
}

func (r *Repository) OpenPendingReconciliation(ctx context.Context, olderThan time.Time, now time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO reconciliation_item(id,payment_id,discrepancy_type,status,created_at) SELECT gen_random_uuid(),p.id,'PENDING_PROVIDER_RESULT','OPEN',$2 FROM payment p WHERE p.state='PENDING' AND p.updated_at<=$1 AND NOT EXISTS(SELECT 1 FROM reconciliation_item r WHERE r.payment_id=p.id AND r.discrepancy_type='PENDING_PROVIDER_RESULT' AND r.status='OPEN')`, olderThan, now)
	return tag.RowsAffected(), err
}

func scanPayment(row pgx.Row) (domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(&p.ID, &p.BookingID, &p.AccountID, &p.OrderID, &p.Amount, &p.Currency, &p.Provider, &p.ProviderPaymentID, &p.State, &p.Version, &p.CreatedAt, &p.UpdatedAt, &p.CompletedAt)
	return p, err
}

const columns = `id,booking_id,account_id,order_id,amount::text,currency,provider,COALESCE(provider_payment_id,''),state,version,created_at,updated_at,completed_at`

func (r *Repository) Initiate(ctx context.Context, p domain.Payment, key, fingerprint string) (domain.Payment, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return p, false, err
	}
	defer tx.Rollback(ctx)
	var existingFP, paymentID string
	err = tx.QueryRow(ctx, `SELECT request_fingerprint,payment_id FROM idempotency_record WHERE actor_id=$1 AND operation='payment.initiate' AND idempotency_key=$2`, p.AccountID, key).Scan(&existingFP, &paymentID)
	if err == nil {
		if existingFP != fingerprint {
			return p, false, application.ErrConflict
		}
		existing, findErr := scanPayment(tx.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE id=$1`, paymentID))
		return existing, true, findErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, false, err
	}
	existing, findErr := scanPayment(tx.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE booking_id=$1 AND account_id=$2`, p.BookingID, p.AccountID))
	if findErr == nil {
		_, err = tx.Exec(ctx, `INSERT INTO idempotency_record(actor_id,operation,idempotency_key,request_fingerprint,payment_id,created_at,expires_at) VALUES($1,'payment.initiate',$2,$3,$4,$5,$5::timestamptz+interval '24 hours')`, p.AccountID, key, fingerprint, existing.ID, p.CreatedAt)
		if err != nil {
			return p, false, err
		}
		if err = tx.Commit(ctx); err != nil {
			return p, false, err
		}
		return existing, true, nil
	}
	if !errors.Is(findErr, pgx.ErrNoRows) {
		return p, false, findErr
	}
	_, err = tx.Exec(ctx, `INSERT INTO payment(id,booking_id,account_id,order_id,amount,currency,provider,state,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,'PAYHERE','PENDING',1,$7,$7)`, p.ID, p.BookingID, p.AccountID, p.OrderID, p.Amount, p.Currency, p.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return p, false, application.ErrConflict
		}
		return p, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_record(actor_id,operation,idempotency_key,request_fingerprint,payment_id,created_at,expires_at) VALUES($1,'payment.initiate',$2,$3,$4,$5,$5::timestamptz+interval '24 hours')`, p.AccountID, key, fingerprint, p.ID, p.CreatedAt)
	if err != nil {
		return p, false, err
	}
	payload, _ := json.Marshal(map[string]string{"paymentId": p.ID, "bookingId": p.BookingID, "amount": p.Amount, "currency": p.Currency})
	_, err = tx.Exec(ctx, `INSERT INTO payment_audit(id,payment_id,action,to_state,actor_id,occurred_at) VALUES($1,$2,'INITIATED','PENDING',$3,$4)`, uuid.NewString(), p.ID, p.AccountID, p.CreatedAt)
	if err != nil {
		return p, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,aggregate_id,correlation_id,actor_id,payload,occurred_at) VALUES($1,'PaymentInitiated',1,$2,$3,$4,$5,$6)`, uuid.NewString(), p.ID, uuid.NewString(), p.AccountID, payload, p.CreatedAt)
	if err != nil {
		return p, false, err
	}
	return p, false, tx.Commit(ctx)
}

func (r *Repository) FindByBooking(ctx context.Context, actor, booking string) (domain.Payment, error) {
	p, err := scanPayment(r.pool.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE booking_id=$1 AND account_id=$2`, booking, actor))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, application.ErrNotFound
	}
	return p, err
}

func (r *Repository) ApplyCallback(ctx context.Context, n payhere.Notification, verification string, now time.Time) (domain.Payment, bool, error) {
	raw := n.MerchantID + "|" + n.OrderID + "|" + n.PaymentID + "|" + n.Amount + "|" + n.Currency + "|" + n.StatusCode + "|" + n.Signature
	s := sha256.Sum256([]byte(raw))
	fingerprint := hex.EncodeToString(s[:])
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Payment{}, false, err
	}
	defer tx.Rollback(ctx)
	p, err := scanPayment(tx.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE order_id=$1 FOR UPDATE`, n.OrderID))
	if errors.Is(err, pgx.ErrNoRows) {
		_, e := tx.Exec(ctx, `INSERT INTO provider_callback(id,fingerprint,order_id,provider_payment_id,verification_result,received_at) VALUES($1,$2,$3,$4,'REVIEW_UNKNOWN_ORDER',$5) ON CONFLICT(fingerprint) DO NOTHING`, uuid.NewString(), fingerprint, n.OrderID, n.PaymentID, now)
		if e != nil {
			return p, false, e
		}
		return p, false, tx.Commit(ctx)
	}
	if err != nil {
		return p, false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO provider_callback(id,fingerprint,payment_id,order_id,provider_payment_id,verification_result,received_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(fingerprint) DO NOTHING`, uuid.NewString(), fingerprint, p.ID, n.OrderID, n.PaymentID, verification, now)
	if err != nil {
		return p, false, err
	}
	if tag.RowsAffected() == 0 {
		return p, true, tx.Commit(ctx)
	}
	next := domain.CallbackState(n.StatusCode)
	reason := "PROVIDER_STATUS_" + n.StatusCode
	if verification != "VERIFIED" || n.Amount != p.Amount || n.Currency != p.Currency {
		next = domain.Review
		reason = "CALLBACK_MISMATCH"
	}
	changed := p.State == domain.Pending && next != domain.Pending
	if changed {
		_, err = tx.Exec(ctx, `UPDATE payment SET state=$2,provider_payment_id=NULLIF($3,''),version=version+1,updated_at=$4,completed_at=CASE WHEN $2='COMPLETED' THEN $4 ELSE completed_at END WHERE id=$1`, p.ID, next, n.PaymentID, now)
		if err != nil {
			return p, false, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO payment_audit(id,payment_id,action,from_state,to_state,actor_id,reason_code,occurred_at) VALUES($1,$2,'CALLBACK','PENDING',$3,'system:payhere',$4,$5)`, uuid.NewString(), p.ID, next, reason, now)
		if err != nil {
			return p, false, err
		}
		eventType := "PaymentFailed"
		if next == domain.Completed {
			eventType = "PaymentCompleted"
		}
		if next == domain.Review {
			eventType = "PaymentReviewRequired"
		}
		payload, _ := json.Marshal(map[string]string{"paymentId": p.ID, "bookingId": p.BookingID, "amount": p.Amount, "currency": p.Currency})
		_, err = tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,aggregate_id,correlation_id,actor_id,payload,occurred_at) VALUES($1,$2,1,$3,$4,'system:payhere',$5,$6)`, uuid.NewString(), eventType, p.ID, uuid.NewString(), payload, now)
		if err != nil {
			return p, false, err
		}
		p.State = next
	}
	return p, !changed, tx.Commit(ctx)
}
