package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const consumerName = "notification.business.v1"

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Ready(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) ApplyEvent(ctx context.Context, event domain.EventEnvelope, plan domain.Plan, now time.Time, maxAttempts int) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `INSERT INTO notification_inbox(consumer,event_id,event_type,processed_at) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, consumerName, event.EventID, event.EventType, now)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}

	switch plan.Action {
	case domain.ActionIgnore:
		return false, tx.Commit(ctx)
	case domain.ActionSuppress:
		_, err = tx.Exec(ctx, `INSERT INTO notification_suppression(account_id,reason,created_at) VALUES($1,$2,$3) ON CONFLICT(account_id) DO UPDATE SET reason=EXCLUDED.reason,created_at=EXCLUDED.created_at,expires_at=NULL`, plan.RecipientAccountID, plan.SuppressionReason, now)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `UPDATE notification_delivery SET state='SUPPRESSED',lease_until=NULL,updated_at=$2,last_error_code=$3 WHERE recipient_account_id=$1 AND state IN ('PENDING','RETRY_SCHEDULED','PROCESSING')`, plan.RecipientAccountID, now, plan.SuppressionReason)
		if err != nil {
			return false, err
		}
		return false, tx.Commit(ctx)
	case domain.ActionDeliver:
	default:
		return false, errors.New("unsupported notification action")
	}

	var templateID string
	err = tx.QueryRow(ctx, `SELECT id FROM notification_template WHERE template_key=$1 AND locale=$2 AND channel=$3 AND status='ACTIVE' ORDER BY version DESC LIMIT 1`, plan.TemplateKey, plan.Locale, plan.Channel).Scan(&templateID)
	if err != nil {
		return false, err
	}
	var suppressed bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notification_suppression WHERE account_id=$1 AND (expires_at IS NULL OR expires_at>$2))`, plan.RecipientAccountID, now).Scan(&suppressed); err != nil {
		return false, err
	}
	var preferred bool
	if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT allowed FROM notification_preference WHERE account_id=$1 AND channel=$2 AND category=$3),true)`, plan.RecipientAccountID, plan.Channel, plan.Category).Scan(&preferred); err != nil {
		return false, err
	}
	state := domain.DeliveryPending
	if suppressed || !preferred {
		state = domain.DeliverySuppressed
	}
	variables, err := json.Marshal(plan.Variables)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notification_delivery(id,business_key,recipient_account_id,source_event_id,source_event_type,source_aggregate_id,template_id,category,channel,variables,state,scheduled_at,next_attempt_at,max_attempts,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12,$13,$12,$12) ON CONFLICT(business_key) DO NOTHING`, uuid.NewString(), plan.BusinessKey, plan.RecipientAccountID, event.EventID, event.EventType, plan.SourceAggregateID, templateID, plan.Category, plan.Channel, variables, state, now, maxAttempts)
	if err != nil {
		return false, err
	}
	if state != domain.DeliverySuppressed {
		_, err = tx.Exec(ctx, `INSERT INTO notification_feed_item(id,delivery_id,recipient_account_id,created_at)
SELECT id,id,recipient_account_id,created_at FROM notification_delivery WHERE business_key=$1
ON CONFLICT(delivery_id) DO NOTHING`, plan.BusinessKey)
		if err != nil {
			return false, err
		}
	}
	return state == domain.DeliveryPending, tx.Commit(ctx)
}

func (r *Repository) ListFeed(ctx context.Context, accountID string, limit int, cursor *domain.FeedCursor) ([]domain.FeedRecord, bool, error) {
	arguments := []any{accountID, limit + 1}
	query := `
SELECT f.id,d.source_event_type,d.category,t.id,t.template_key,t.version,t.locale,t.channel,t.category,
       t.subject_template,t.body_template,t.allowed_variables,d.variables,f.read_at,f.created_at
FROM notification_feed_item f
JOIN notification_delivery d ON d.id=f.delivery_id
JOIN notification_template t ON t.id=d.template_id
WHERE f.recipient_account_id=$1`
	if cursor != nil {
		query += ` AND (f.created_at,f.id)<($3,$4)`
		arguments = append(arguments, cursor.CreatedAt, cursor.ID)
	}
	query += ` ORDER BY f.created_at DESC,f.id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	records := make([]domain.FeedRecord, 0, limit+1)
	for rows.Next() {
		var record domain.FeedRecord
		var variables []byte
		if err = rows.Scan(
			&record.ID, &record.SourceEventType, &record.Category,
			&record.Template.ID, &record.Template.Key, &record.Template.Version, &record.Template.Locale,
			&record.Template.Channel, &record.Template.Category, &record.Template.SubjectTemplate,
			&record.Template.BodyTemplate, &record.Template.AllowedVariables, &variables,
			&record.ReadAt, &record.CreatedAt,
		); err != nil {
			return nil, false, err
		}
		if err = json.Unmarshal(variables, &record.Variables); err != nil {
			return nil, false, err
		}
		records = append(records, record)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	return records, hasMore, nil
}

func (r *Repository) UnreadCount(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notification_feed_item WHERE recipient_account_id=$1 AND read_at IS NULL`, accountID).Scan(&count)
	return count, err
}

func (r *Repository) MarkRead(ctx context.Context, accountID, notificationID string, now time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE notification_feed_item SET read_at=COALESCE(read_at,$3) WHERE id=$1 AND recipient_account_id=$2`, notificationID, accountID, now)
	return tag.RowsAffected() == 1, err
}

func (r *Repository) MarkAllRead(ctx context.Context, accountID string, now time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE notification_feed_item SET read_at=$2 WHERE recipient_account_id=$1 AND read_at IS NULL`, accountID, now)
	return tag.RowsAffected(), err
}

func (r *Repository) ClaimDue(ctx context.Context, now time.Time, lease time.Duration) (domain.Delivery, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Delivery{}, false, err
	}
	defer tx.Rollback(ctx)

	var delivery domain.Delivery
	var variables []byte
	err = tx.QueryRow(ctx, `
SELECT d.id,d.business_key,d.recipient_account_id,d.source_event_id,d.source_event_type,d.source_aggregate_id,
       d.state,d.attempt_count,d.max_attempts,d.created_at,d.updated_at,d.variables,
       t.id,t.template_key,t.version,t.locale,t.channel,t.category,t.subject_template,t.body_template,t.allowed_variables
FROM notification_delivery d
JOIN notification_template t ON t.id=d.template_id
WHERE ((d.state IN ('PENDING','RETRY_SCHEDULED') AND d.next_attempt_at<=$1)
       OR (d.state='PROCESSING' AND d.lease_until<=$1))
ORDER BY d.next_attempt_at,d.created_at
FOR UPDATE OF d SKIP LOCKED
LIMIT 1`, now).Scan(
		&delivery.ID, &delivery.BusinessKey, &delivery.RecipientAccountID, &delivery.SourceEventID, &delivery.SourceEventType, &delivery.SourceAggregateID,
		&delivery.State, &delivery.AttemptCount, &delivery.MaxAttempts, &delivery.CreatedAt, &delivery.UpdatedAt, &variables,
		&delivery.Template.ID, &delivery.Template.Key, &delivery.Template.Version, &delivery.Template.Locale, &delivery.Template.Channel, &delivery.Template.Category,
		&delivery.Template.SubjectTemplate, &delivery.Template.BodyTemplate, &delivery.Template.AllowedVariables,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Delivery{}, false, nil
	}
	if err != nil {
		return domain.Delivery{}, false, err
	}
	if err = json.Unmarshal(variables, &delivery.Variables); err != nil {
		return domain.Delivery{}, false, err
	}
	delivery.AttemptCount++
	delivery.State = domain.DeliveryProcessing
	delivery.UpdatedAt = now
	if _, err = tx.Exec(ctx, `UPDATE notification_delivery SET state='PROCESSING',attempt_count=$2,lease_until=$3,updated_at=$4 WHERE id=$1`, delivery.ID, delivery.AttemptCount, now.Add(lease), now); err != nil {
		return domain.Delivery{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Delivery{}, false, err
	}
	return delivery, true, nil
}

type AttemptResult struct {
	Delivered         bool
	PermanentFailure  bool
	ProviderReference string
	ErrorCode         string
	StartedAt         time.Time
	CompletedAt       time.Time
	RetryAt           time.Time
}

func (r *Repository) CompleteAttempt(ctx context.Context, delivery domain.Delivery, result AttemptResult) (domain.DeliveryState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var state domain.DeliveryState
	var attemptCount, maxAttempts int
	err = tx.QueryRow(ctx, `SELECT state,attempt_count,max_attempts FROM notification_delivery WHERE id=$1 FOR UPDATE`, delivery.ID).Scan(&state, &attemptCount, &maxAttempts)
	if err != nil {
		return "", err
	}
	if state != domain.DeliveryProcessing || attemptCount != delivery.AttemptCount {
		return state, errors.New("notification delivery lease is stale")
	}

	nextState := domain.DeliveryRetryScheduled
	outcome := "RETRYABLE_FAILURE"
	if result.Delivered {
		nextState = domain.DeliveryDelivered
		outcome = "DELIVERED"
	} else if result.PermanentFailure {
		nextState = domain.DeliveryPermanentlyFailed
		outcome = "PERMANENT_FAILURE"
	} else if attemptCount >= maxAttempts {
		nextState = domain.DeliveryDeadLettered
		outcome = "DEAD_LETTERED"
	}

	_, err = tx.Exec(ctx, `INSERT INTO notification_delivery_attempt(id,delivery_id,attempt_number,outcome,provider_reference,error_code,started_at,completed_at) VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,$8)`, uuid.NewString(), delivery.ID, attemptCount, outcome, result.ProviderReference, result.ErrorCode, result.StartedAt, result.CompletedAt)
	if err != nil {
		return "", err
	}

	nextAttempt := result.RetryAt
	if nextAttempt.IsZero() {
		nextAttempt = result.CompletedAt
	}
	_, err = tx.Exec(ctx, `UPDATE notification_delivery SET state=$2,next_attempt_at=$3,provider_reference=NULLIF($4,''),last_error_code=NULLIF($5,''),lease_until=NULL,updated_at=$6,delivered_at=CASE WHEN $2='DELIVERED' THEN $6 ELSE delivered_at END WHERE id=$1`, delivery.ID, nextState, nextAttempt, result.ProviderReference, result.ErrorCode, result.CompletedAt)
	if err != nil {
		return "", err
	}
	return nextState, tx.Commit(ctx)
}
