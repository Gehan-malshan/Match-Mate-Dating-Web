package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) staffAction(ctx context.Context, actor, bookingID, key, operation, method string, now time.Time) (domain.Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Booking{}, err
	}
	defer tx.Rollback(ctx)
	fingerprint := bookingID + ":" + method
	var priorID, priorFingerprint string
	err = tx.QueryRow(ctx, `SELECT booking_id::text,request_fingerprint FROM booking_staff_action WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3`, actor, operation, key).Scan(&priorID, &priorFingerprint)
	if err == nil {
		if priorID != bookingID || priorFingerprint != fingerprint {
			return domain.Booking{}, application.ErrConflict
		}
		return scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1`, bookingID))
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, err
	}
	b, err := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1 FOR UPDATE`, bookingID))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, application.ErrNotFound
	}
	if err != nil {
		return b, err
	}
	// A retry can commit while this transaction waits for the booking row lock.
	err = tx.QueryRow(ctx, `SELECT booking_id::text,request_fingerprint FROM booking_staff_action WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3`, actor, operation, key).Scan(&priorID, &priorFingerprint)
	if err == nil {
		if priorID != bookingID || priorFingerprint != fingerprint {
			return b, application.ErrConflict
		}
		return b, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return b, err
	}
	if b.State != domain.Confirmed || b.CheckedInAt != nil {
		return b, application.ErrConflict
	}
	switch operation {
	case "venue.collect":
		if b.PaymentMethod != "AT_VENUE" || b.VenueCollectedAt != nil || (method != "CASH" && method != "CARD_AT_VENUE") {
			return b, application.ErrConflict
		}
		_, err = tx.Exec(ctx, `UPDATE booking SET venue_collected_at=$2,venue_collected_by=$3,venue_collection_method=$4,version=version+1 WHERE id=$1`, bookingID, now, actor, method)
		b.VenueCollectedAt = &now
		b.VenueCollectionMethod = &method
	case "booking.check_in":
		if b.PaymentMethod == "AT_VENUE" && b.VenueCollectedAt == nil {
			return b, application.ErrConflict
		}
		_, err = tx.Exec(ctx, `UPDATE booking SET checked_in_at=$2,checked_in_by=$3,version=version+1 WHERE id=$1`, bookingID, now, actor)
		b.CheckedInAt = &now
	default:
		return b, application.ErrInvalid
	}
	if err != nil {
		return b, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO booking_staff_action(actor_id,operation,idempotency_key,booking_id,request_fingerprint,created_at) VALUES($1,$2,$3,$4,$5,$6)`, actor, operation, key, bookingID, fingerprint, now)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return b, application.ErrConflict
		}
		return b, err
	}
	b.Version++
	eventType := "BookingCheckedIn"
	if operation == "venue.collect" {
		eventType = "VenuePaymentCollected"
	}
	if err = insertFact(ctx, tx, eventType, b, "", actor, now); err != nil {
		return b, err
	}
	if err = tx.Commit(ctx); err != nil {
		return b, err
	}
	return b, nil
}

func (r *Repository) CollectVenue(ctx context.Context, actor, bookingID, key, method string, now time.Time) (domain.Booking, error) {
	return r.staffAction(ctx, actor, bookingID, key, "venue.collect", method, now)
}
func (r *Repository) GetForStaff(ctx context.Context, bookingID string) (domain.Booking, error) {
	b, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM booking WHERE id=$1`, bookingID))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, application.ErrNotFound
	}
	return b, err
}
func (r *Repository) CheckIn(ctx context.Context, actor, bookingID, key string, now time.Time) (domain.Booking, error) {
	return r.staffAction(ctx, actor, bookingID, key, "booking.check_in", "", now)
}
