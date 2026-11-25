package postgres_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/application"
	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
	storepg "github.com/gehan-malshan/matchmate/payment-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/payment-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPaymentPersistenceAndCallbackRecovery(t *testing.T) {
	url := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set PAYMENT_TEST_DATABASE_URL to run PostgreSQL component tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "payment_test_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	if _, err = admin.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema)) })
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sql, err := migrations.Files.ReadFile("000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	repo := storepg.New(pool)
	now := time.Now().UTC()
	actor, booking := uuid.NewString(), uuid.NewString()
	newPayment := func(bookingID string) domain.Payment {
		return domain.Payment{ID: uuid.NewString(), BookingID: bookingID, AccountID: actor, OrderID: "MM-" + uuid.NewString(), Amount: "2500.00", Currency: "LKR", Provider: "PAYHERE", State: domain.Pending, Version: 1, CreatedAt: now, UpdatedAt: now}
	}
	var wg sync.WaitGroup
	results := make(chan domain.Payment, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, _, e := repo.Initiate(ctx, newPayment(booking), "same-key", "same-fingerprint")
			results <- p
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var payment domain.Payment
	for p := range results {
		if payment.ID != "" && payment.ID != p.ID {
			t.Fatal("retry created a second payment")
		}
		payment = p
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM payment WHERE booking_id=$1`, booking).Scan(&count); err != nil || count != 1 {
		t.Fatalf("payment count %d: %v", count, err)
	}
	provider := payhere.Config{MerchantID: "123", MerchantSecret: "secret"}
	svc := application.New(repo, nil, provider)
	n := payhere.Notification{MerchantID: "123", OrderID: payment.OrderID, PaymentID: "provider-1", Amount: payment.Amount, Currency: payment.Currency, StatusCode: "2", Signature: "bad"}
	if err = svc.Callback(ctx, n); err == nil {
		t.Fatal("invalid signature accepted")
	}
	opened, err := repo.OpenPendingReconciliation(ctx, now.Add(time.Minute), now.Add(time.Minute))
	if err != nil || opened != 1 {
		t.Fatalf("pending reconciliation was not opened: count=%d err=%v", opened, err)
	}
	n.Signature = payhere.NotificationHash(n, provider.MerchantSecret)
	for i := 0; i < 2; i++ {
		if err = svc.Callback(ctx, n); err != nil {
			t.Fatal(err)
		}
	}
	stored, err := repo.FindByBooking(ctx, actor, booking)
	if err != nil || stored.State != domain.Completed {
		t.Fatalf("verified payment not completed: %+v %v", stored, err)
	}
	if _, _, err = repo.Initiate(ctx, newPayment(booking), "another-key", "same-fingerprint"); err == nil {
		t.Fatal("completed payment offered a second checkout")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE event_type='PaymentCompleted' AND aggregate_id=$1`, payment.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("completion events %d: %v", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM provider_callback WHERE payment_id=$1`, payment.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("callback receipts %d: %v", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM reconciliation_item WHERE payment_id=$1 AND status='OPEN'`, payment.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("resolved payment still pending review: %d %v", count, err)
	}
	chargeback := n
	chargeback.StatusCode = "-3"
	chargeback.Signature = payhere.NotificationHash(chargeback, provider.MerchantSecret)
	if err = svc.Callback(ctx, chargeback); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.FindByBooking(ctx, actor, booking)
	if err != nil || stored.State != domain.Review {
		t.Fatalf("chargeback was not flagged for review: %+v %v", stored, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM reconciliation_item WHERE payment_id=$1 AND status='OPEN'`, payment.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("chargeback reconciliation items %d: %v", count, err)
	}
	review, err := repo.ListReview(ctx, actor, 20, "")
	if err != nil || len(review.Items) != 1 || review.Items[0].PaymentID != payment.ID || review.Items[0].Reason != "CONFLICTING_PROVIDER_RESULT" {
		t.Fatalf("review queue mismatch: %+v %v", review, err)
	}
	otherBooking := uuid.NewString()
	other, _, err := repo.Initiate(ctx, newPayment(otherBooking), "other-key", "other-fingerprint")
	if err != nil {
		t.Fatal(err)
	}
	mismatch := payhere.Notification{MerchantID: "123", OrderID: other.OrderID, PaymentID: "provider-2", Amount: "1.00", Currency: other.Currency, StatusCode: "2"}
	mismatch.Signature = payhere.NotificationHash(mismatch, provider.MerchantSecret)
	if err = svc.Callback(ctx, mismatch); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.FindByBooking(ctx, actor, otherBooking)
	if err != nil || stored.State != domain.Review {
		t.Fatalf("amount mismatch escaped review: %+v %v", stored, err)
	}
	unknown := payhere.Notification{MerchantID: "123", OrderID: "MM-" + uuid.NewString(), PaymentID: "provider-unknown", Amount: "2500.00", Currency: "LKR", StatusCode: "2"}
	unknown.Signature = payhere.NotificationHash(unknown, provider.MerchantSecret)
	if err = svc.Callback(ctx, unknown); err != nil {
		t.Fatal(err)
	}
	first, err := repo.ListReview(ctx, actor, 1, "")
	if err != nil || len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first review page: %+v %v", first, err)
	}
	second, err := repo.ListReview(ctx, actor, 1, first.NextCursor)
	if err != nil || len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("second review page: %+v %v", second, err)
	}
	if _, err = repo.ListReview(ctx, actor, 1, "invalid cursor"); err == nil {
		t.Fatal("invalid review cursor accepted")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM payment_audit WHERE action='RECONCILIATION_VIEW' AND actor_id=$1`, actor).Scan(&count); err != nil || count != 3 {
		t.Fatalf("privileged reads not audited: count=%d err=%v", count, err)
	}
}
