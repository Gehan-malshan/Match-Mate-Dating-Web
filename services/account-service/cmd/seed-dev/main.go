package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/config"
	"github.com/jackc/pgx/v5"
)

const developmentPassword = "MatchMateDev123!"

type seedAccount struct {
	id, email, nickname, status, visibility, approval, role string
}

// communityFixture has no credential. It makes the Community page useful in a
// fresh local environment without adding more accounts that can sign in.
type communityFixture struct {
	id, email, nickname, dateOfBirth, location, bio string
	interests                                       []string
}

var developmentAccounts = []seedAccount{
	{"00000000-0000-4000-8000-000000000001", "member@example.test", "Private Member", "ACTIVE", "PRIVATE", "PENDING", "member"},
	{"00000000-0000-4000-8000-000000000002", "community@example.test", "Community Member", "ACTIVE", "COMMUNITY", "APPROVED", "member"},
	{"00000000-0000-4000-8000-000000000003", "moderator@example.test", "Test Moderator", "ACTIVE", "PRIVATE", "APPROVED", "moderator"},
	{"00000000-0000-4000-8000-000000000004", "suspended@example.test", "Suspended Member", "SUSPENDED", "HIDDEN", "HIDDEN", "member"},
	{"00000000-0000-4000-8000-000000000005", "organizer@example.test", "Test Organizer", "ACTIVE", "PRIVATE", "APPROVED", "organizer"},
}

var developmentCommunityProfiles = []communityFixture{
	{"10000000-0000-4000-8000-000000000001", "maya.fixture@example.test", "Maya", "1997-04-18", "Colombo 05", "A warm listener who enjoys slow mornings, good design, and discovering new places around the city.", []string{"Architecture", "Coffee", "Weekend walks"}},
	{"10000000-0000-4000-8000-000000000002", "aki.fixture@example.test", "Aki", "1994-11-03", "Colombo 07", "Here for thoughtful conversation, live music, and the kind of connection that grows at its own pace.", []string{"Live music", "Photography", "Sri Lankan food"}},
	{"10000000-0000-4000-8000-000000000003", "savi.fixture@example.test", "Savi", "1998-06-22", "Nugegoda", "Curious by nature and happiest when a plan includes books, a gallery, or an unhurried conversation.", []string{"Books", "Art", "Cooking"}},
	{"10000000-0000-4000-8000-000000000004", "nila.fixture@example.test", "Nila", "1995-09-14", "Rajagiriya", "A calm, optimistic person who values kindness, consistency, and making ordinary weekends feel special.", []string{"Film", "Hiking", "Board games"}},
	{"10000000-0000-4000-8000-000000000005", "kavi.fixture@example.test", "Kavi", "1993-02-09", "Colombo 03", "Enjoys curious questions, small dinner tables, and building a life with room for laughter and ambition.", []string{"Travel", "Design", "Tennis"}},
	{"10000000-0000-4000-8000-000000000006", "rhea.fixture@example.test", "Rhea", "1996-12-01", "Mount Lavinia", "A fan of sunset walks, thoughtful podcasts, and meeting people who are comfortable being themselves.", []string{"Podcasts", "Beach walks", "Yoga"}},
}

func main() {
	environment := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if err := ensureDevelopment(environment); err != nil {
		panic(err)
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	passwordHash, err := auth.HashPassword(developmentPassword)
	if err != nil {
		panic(err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	for _, account := range developmentAccounts {
		if err = seed(ctx, tx, account, passwordHash, now, cfg.CurrentConsentVersion); err != nil {
			panic(err)
		}
	}
	for _, profile := range developmentCommunityProfiles {
		if err = seedCommunityFixture(ctx, tx, profile, now, cfg.CurrentConsentVersion); err != nil {
			panic(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		panic(err)
	}

	fmt.Println("Development accounts are ready. This password is public test data only:")
	for _, account := range developmentAccounts {
		fmt.Printf("  %-26s role=%-9s status=%s\n", account.email, account.role, account.status)
	}
	fmt.Printf("  password: %s\n", developmentPassword)
}

func seedCommunityFixture(ctx context.Context, tx pgx.Tx, profile communityFixture, now time.Time, consentVersion string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO account(account_id,normalized_email,status,verification_state,token_version,created_at,updated_at) VALUES($1,$2,'ACTIVE','VERIFIED',1,$3,$3) ON CONFLICT(account_id) DO UPDATE SET normalized_email=EXCLUDED.normalized_email,status='ACTIVE',verification_state='VERIFIED',updated_at=EXCLUDED.updated_at`, profile.id, profile.email, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO consent_record(consent_id,account_id,consent_type,policy_version,granted_at) VALUES($1,$2,'privacy-and-terms',$3,$4) ON CONFLICT(consent_id) DO UPDATE SET policy_version=EXCLUDED.policy_version,granted_at=EXCLUDED.granted_at,revoked_at=NULL`, strings.Replace(profile.id, "4000", "4100", 1), profile.id, consentVersion, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO profile(account_id,nickname,date_of_birth,broad_location,bio,visibility,approval_state,version,updated_at) VALUES($1,$2,$3,$4,$5,'COMMUNITY','APPROVED',1,$6) ON CONFLICT(account_id) DO UPDATE SET nickname=EXCLUDED.nickname,date_of_birth=EXCLUDED.date_of_birth,broad_location=EXCLUDED.broad_location,bio=EXCLUDED.bio,visibility='COMMUNITY',approval_state='APPROVED',version=profile.version+1,updated_at=EXCLUDED.updated_at`, profile.id, profile.nickname, profile.dateOfBirth, profile.location, profile.bio, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignment(account_id,role,granted_at) VALUES($1,'member',$2) ON CONFLICT(account_id,role,scope) DO UPDATE SET revoked_at=NULL`, profile.id, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM profile_interest WHERE account_id=$1`, profile.id); err != nil {
		return err
	}
	for _, interest := range profile.interests {
		if _, err := tx.Exec(ctx, `INSERT INTO profile_interest(account_id,interest) VALUES($1,$2)`, profile.id, interest); err != nil {
			return err
		}
	}
	return nil
}

func ensureDevelopment(environment string) error {
	if environment != "development" && environment != "test" {
		return errors.New("seed-dev is allowed only when APP_ENV is development or test")
	}
	return nil
}

func seed(ctx context.Context, tx pgx.Tx, account seedAccount, passwordHash string, now time.Time, consentVersion string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO account(account_id,normalized_email,status,verification_state,token_version,created_at,updated_at)
		VALUES($1,$2,$3,'VERIFIED',1,$4,$4)
		ON CONFLICT(account_id) DO UPDATE SET normalized_email=EXCLUDED.normalized_email,status=EXCLUDED.status,verification_state='VERIFIED',token_version=account.token_version+1,updated_at=EXCLUDED.updated_at`, account.id, account.email, account.status, now)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE refresh_session SET revoked_at=COALESCE(revoked_at,$2) WHERE account_id=$1`, account.id, now); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO credential(account_id,password_hash,changed_at) VALUES($1,$2,$3) ON CONFLICT(account_id) DO UPDATE SET password_hash=EXCLUDED.password_hash,changed_at=EXCLUDED.changed_at`, account.id, passwordHash, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consent_record(consent_id,account_id,consent_type,policy_version,granted_at) VALUES($1,$2,'privacy-and-terms',$3,$4) ON CONFLICT(consent_id) DO UPDATE SET policy_version=EXCLUDED.policy_version,granted_at=EXCLUDED.granted_at,revoked_at=NULL`, strings.Replace(account.id, "4000", "4100", 1), account.id, consentVersion, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO profile(account_id,nickname,date_of_birth,broad_location,bio,visibility,approval_state,version,updated_at) VALUES($1,$2,'1995-01-15','Colombo','Development-only MatchMate test profile.',$3,$4,1,$5) ON CONFLICT(account_id) DO UPDATE SET nickname=EXCLUDED.nickname,broad_location=EXCLUDED.broad_location,bio=EXCLUDED.bio,visibility=EXCLUDED.visibility,approval_state=EXCLUDED.approval_state,version=profile.version+1,updated_at=EXCLUDED.updated_at`, account.id, account.nickname, account.visibility, account.approval, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO role_assignment(account_id,role,granted_at) VALUES($1,'member',$2) ON CONFLICT(account_id,role,scope) DO UPDATE SET revoked_at=NULL`, account.id, now)
	if err != nil {
		return err
	}
	if account.role != "member" {
		_, err = tx.Exec(ctx, `INSERT INTO role_assignment(account_id,role,granted_at) VALUES($1,$2,$3) ON CONFLICT(account_id,role,scope) DO UPDATE SET revoked_at=NULL`, account.id, account.role, now)
		if err != nil {
			return err
		}
	}
	if account.email == "community@example.test" {
		for _, interest := range []string{"Books", "Music", "Travel"} {
			if _, err = tx.Exec(ctx, `INSERT INTO profile_interest(account_id,interest) VALUES($1,$2) ON CONFLICT DO NOTHING`, account.id, interest); err != nil {
				return err
			}
		}
	}
	return nil
}
