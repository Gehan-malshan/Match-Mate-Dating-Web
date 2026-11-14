package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
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

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

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
	tag, err := r.pool.Exec(ctx, `WITH due AS (
		SELECT p.id FROM payment p WHERE p.state='PENDING' AND p.updated_at<=$1
		AND NOT EXISTS(SELECT 1 FROM reconciliation_item r WHERE r.payment_id=p.id AND r.discrepancy_type='PENDING_PROVIDER_RESULT' AND r.status='OPEN')
		FOR UPDATE OF p SKIP LOCKED
	) INSERT INTO reconciliation_item(id,payment_id,discrepancy_type,status,created_at)
	SELECT gen_random_uuid(),due.id,'PENDING_PROVIDER_RESULT','OPEN',$2 FROM due ON CONFLICT DO NOTHING`, olderThan, now)
	return tag.RowsAffected(), err
}

func (r *Repository) ListReview(ctx context.Context, actor string, limit int, cursor string) (application.ReviewPage, error) {
	var before *time.Time
	var beforeID string
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return application.ReviewPage{}, application.ErrInvalidCursor
		}
		parts := strings.SplitN(string(decoded), "|", 2)
		if len(parts) != 2 {
			return application.ReviewPage{}, application.ErrInvalidCursor
		}
		at, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return application.ReviewPage{}, application.ErrInvalidCursor
		}
		if _, err = uuid.Parse(parts[1]); err != nil {
			return application.ReviewPage{}, application.ErrInvalidCursor
		}
		before, beforeID = &at, parts[1]
	}
	rows, err := r.pool.Query(ctx, `SELECT id,payment_id,booking_id,order_id,amount,currency,state,reason,created_at FROM (
		SELECT r.id::text AS id,p.id::text AS payment_id,p.booking_id::text AS booking_id,p.order_id,p.amount::text AS amount,p.currency::text AS currency,p.state,r.discrepancy_type AS reason,r.created_at
		FROM reconciliation_item r JOIN payment p ON p.id=r.payment_id WHERE r.status='OPEN'
		UNION ALL
		SELECT c.id::text,'','',c.order_id,'','','UNKNOWN_ORDER',c.verification_result,c.received_at
		FROM provider_callback c WHERE c.payment_id IS NULL AND c.verification_result='REVIEW_UNKNOWN_ORDER'
	) items WHERE ($2::timestamptz IS NULL OR (created_at,id)<($2::timestamptz,$3::text)) ORDER BY created_at DESC,id DESC LIMIT $1`, limit+1, before, beforeID)
	if err != nil {
		return application.ReviewPage{}, err
	}
	defer rows.Close()
	items := []application.ReviewItem{}
	for rows.Next() {
		var item application.ReviewItem
		if err = rows.Scan(&item.ID, &item.PaymentID, &item.BookingID, &item.OrderID, &item.Amount, &item.Currency, &item.State, &item.Reason, &item.CreatedAt); err != nil {
			return application.ReviewPage{}, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return application.ReviewPage{}, err
	}
	rows.Close()
	page := application.ReviewPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID))
	}
	if _, err = r.pool.Exec(ctx, `INSERT INTO payment_audit(id,action,actor_id,occurred_at) VALUES($1,'RECONCILIATION_VIEW',$2,now())`, uuid.NewString(), actor); err != nil {
		return application.ReviewPage{}, err
	}
	return page, nil
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
	// Serialize both identical-key retries and different-key attempts for one booking.
	// The unique constraints remain the final guard, but retries should observe the
	// committed payment instead of failing on a concurrent insert.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, p.BookingID); err != nil {
		return p, false, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,1))`, p.AccountID+"|"+key); err != nil {
		return p, false, err
	}
	var existingFP, paymentID string
	err = tx.QueryRow(ctx, `SELECT request_fingerprint,payment_id FROM idempotency_record WHERE actor_id=$1 AND operation='payment.initiate' AND idempotency_key=$2`, p.AccountID, key).Scan(&existingFP, &paymentID)
	if err == nil {
		if existingFP != fingerprint {
			return p, false, application.ErrConflict
		}
		existing, findErr := scanPayment(tx.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE id=$1`, paymentID))
		if findErr == nil && existing.State != domain.Pending {
			return p, false, application.ErrConflict
		}
		return existing, true, findErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, false, err
	}
	existing, findErr := scanPayment(tx.QueryRow(ctx, `SELECT `+columns+` FROM payment WHERE booking_id=$1 AND account_id=$2`, p.BookingID, p.AccountID))
	if findErr == nil {
		if existing.State != domain.Pending {
			return p, false, application.ErrConflict
		}
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
	if verification != "VERIFIED" || n.Amount != p.Amount || n.Currency != p.Currency || (p.ProviderPaymentID != "" && p.ProviderPaymentID != n.PaymentID) {
		next = domain.Review
		reason = "CALLBACK_MISMATCH"
	}
	// A later verified chargeback must be visible to finance. Never silently
	// replace a completed booking or treat the callback as an automatic refund.
	if p.State == domain.Completed && (n.StatusCode == "-1" || n.StatusCode == "-2" || n.StatusCode == "-3") {
		next = domain.Review
		reason = "CONFLICTING_PROVIDER_RESULT"
	}
	if p.State == domain.Failed && n.StatusCode == "2" {
		next = domain.Review
		reason = "CONFLICTING_PROVIDER_RESULT"
	}
	changed := (p.State == domain.Pending && next != domain.Pending) || ((p.State == domain.Completed || p.State == domain.Failed) && next == domain.Review)
	if changed {
		_, err = tx.Exec(ctx, `UPDATE payment SET state=$2,provider_payment_id=COALESCE(provider_payment_id,NULLIF($3,'')),version=version+1,updated_at=$4,completed_at=CASE WHEN $2='COMPLETED' THEN $4 ELSE completed_at END WHERE id=$1`, p.ID, next, n.PaymentID, now)
		if err != nil {
			return p, false, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO payment_audit(id,payment_id,action,from_state,to_state,actor_id,reason_code,occurred_at) VALUES($1,$2,'CALLBACK',$3,$4,'system:payhere',$5,$6)`, uuid.NewString(), p.ID, p.State, next, reason, now)
		if err != nil {
			return p, false, err
		}
		_, err = tx.Exec(ctx, `UPDATE reconciliation_item SET status='RESOLVED',resolved_at=$2 WHERE payment_id=$1 AND discrepancy_type='PENDING_PROVIDER_RESULT' AND status='OPEN'`, p.ID, now)
		if err != nil {
			return p, false, err
		}
		if next == domain.Review {
			_, err = tx.Exec(ctx, `INSERT INTO reconciliation_item(id,payment_id,discrepancy_type,status,created_at) VALUES($1,$2,$3,'OPEN',$4) ON CONFLICT DO NOTHING`, uuid.NewString(), p.ID, reason, now)
			if err != nil {
				return p, false, err
			}
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
