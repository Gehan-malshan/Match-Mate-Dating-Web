package postgres

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testRepository(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("ACCOUNT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ACCOUNT_TEST_DATABASE_URL to a disposable migrated database")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return New(pool), pool
}

func TestRateWindowIsAtomicAcrossConcurrentRequests(t *testing.T) {
	repo, _ := testRepository(t)
	key := sha256.Sum256([]byte(uuid.NewString()))
	now := time.Now().UTC()
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.AllowRate(context.Background(), key[:], now, 8)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if accepted != 8 {
		t.Fatalf("accepted %d requests; expected 8", accepted)
	}
}

func TestPasswordResetRevokesSessionsAndDeletionMinimizesAccountData(t *testing.T) {
	repo, pool := testRepository(t)
	ctx := context.Background()
	id := uuid.NewString()
	email := "member-" + id + "@example.test"
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `INSERT INTO account(account_id,normalized_email,status,verification_state,created_at,updated_at) VALUES($1,$2,'ACTIVE','VERIFIED',$3,$3)`, id, email, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO credential(account_id,password_hash,changed_at) VALUES($1,'old',$2)`, id, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profile(account_id,nickname,date_of_birth,visibility,approval_state,updated_at) VALUES($1,'Private Name','1995-01-01','COMMUNITY','APPROVED',$2)`, id, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profile_interest(account_id,interest) VALUES($1,'Walking')`, id)
	if err != nil {
		t.Fatal(err)
	}
	oldSession := uuid.NewString()
	_, err = pool.Exec(ctx, `INSERT INTO refresh_session(session_id,family_id,account_id,token_hash,expires_at,created_at) VALUES($1,$2,$3,$4,$5,$6)`, oldSession, uuid.NewString(), id, []byte(uuid.NewString()), now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	token := sha256.Sum256([]byte(uuid.NewString()))
	if err = repo.CreatePasswordReset(ctx, id, token[:], now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err = repo.ConsumePasswordReset(ctx, token[:], "new-hash", now); err != nil {
		t.Fatal(err)
	}
	if err = repo.ConsumePasswordReset(ctx, token[:], "other", now); err == nil {
		t.Fatal("reset token replayed")
	}
	var hash string
	var version int
	var revoked *time.Time
	if err = pool.QueryRow(ctx, `SELECT c.password_hash,a.token_version,s.revoked_at FROM credential c JOIN account a USING(account_id) JOIN refresh_session s USING(account_id) WHERE a.account_id=$1`, id).Scan(&hash, &version, &revoked); err != nil {
		t.Fatal(err)
	}
	if hash != "new-hash" || version != 2 || revoked == nil {
		t.Fatal("password reset did not revoke sessions")
	}
	event := domain.Event{EventID: uuid.NewString(), EventType: "AccountDeactivated", SchemaVersion: 1, OccurredAt: now, CorrelationID: uuid.NewString(), ActorID: id}
	if err = repo.Deactivate(ctx, id, event); err != nil {
		t.Fatal(err)
	}
	var storedEmail, status string
	var profileCount, credentialCount int
	if err = pool.QueryRow(ctx, `SELECT normalized_email,status FROM account WHERE account_id=$1`, id).Scan(&storedEmail, &status); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM profile WHERE account_id=$1`, id).Scan(&profileCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM credential WHERE account_id=$1`, id).Scan(&credentialCount); err != nil {
		t.Fatal(err)
	}
	if status != "DEACTIVATED" || storedEmail == email || profileCount != 0 || credentialCount != 0 {
		t.Fatal("account-owned personal data not minimized")
	}
}

func TestAdministratorTOTPReplayAndRefreshStrength(t *testing.T) {
	repo, pool := testRepository(t)
	ctx := context.Background()
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `INSERT INTO account(account_id,normalized_email,status,verification_state,created_at,updated_at) VALUES($1,$2,'ACTIVE','VERIFIED',$3,$3)`, id, "admin-"+id+"@example.test", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO role_assignment(account_id,role,granted_at) VALUES($1,'admin',$2)`, id, now)
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	ciphertext, err := auth.EncryptMFASecret(key, "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO admin_totp(account_id,secret_ciphertext,confirmed_at,created_at) VALUES($1,$2,$3,$3)`, id, ciphertext, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.VerifyAdminTOTP(ctx, id, "287082", time.Unix(59, 0), key); err != nil {
		t.Fatal(err)
	}
	if err = repo.VerifyAdminTOTP(ctx, id, "287082", time.Unix(59, 0), key); err == nil {
		t.Fatal("TOTP replay accepted")
	}
	oldRaw := []byte(uuid.NewString())
	session := domain.Session{ID: uuid.NewString(), FamilyID: uuid.NewString(), AccountID: id, TokenHash: oldRaw, ExpiresAt: now.Add(time.Hour), MFAVerified: true}
	if err = repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	next := domain.Session{ID: uuid.NewString(), TokenHash: []byte(uuid.NewString()), ExpiresAt: now.Add(time.Hour)}
	a, err := repo.RotateSession(ctx, oldRaw, next, now)
	if err != nil || !a.MFAVerified {
		t.Fatalf("MFA strength lost on refresh: %v", err)
	}
}
