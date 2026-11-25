package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/jackc/pgx/v5"
)

// Restricted operator command. Run through a secrets-managed terminal; never
// place the printed authenticator secret in a ticket, log, or repository.
func main() {
	if len(os.Args) < 3 {
		panic("usage: admin-mfa enroll EMAIL | confirm EMAIL CODE")
	}
	key, err := auth.DecodeMFAKey(os.Getenv("ADMIN_MFA_ENCRYPTION_KEY"))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	email := strings.ToLower(strings.TrimSpace(os.Args[2]))
	var id string
	err = conn.QueryRow(ctx, `SELECT a.account_id FROM account a JOIN role_assignment ra USING(account_id) WHERE a.normalized_email=$1 AND a.status='ACTIVE' AND ra.role='admin' AND ra.revoked_at IS NULL`, email).Scan(&id)
	if err != nil {
		panic(errors.New("active administrator not found"))
	}
	switch os.Args[1] {
	case "enroll":
		secretBytes := make([]byte, 20)
		if _, err = rand.Read(secretBytes); err != nil {
			panic(err)
		}
		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
		ciphertext, err := auth.EncryptMFASecret(key, secret)
		if err != nil {
			panic(err)
		}
		tag, err := conn.Exec(ctx, `INSERT INTO admin_totp(account_id,secret_ciphertext,created_at) VALUES($1,$2,now()) ON CONFLICT DO NOTHING`, id, ciphertext)
		if err != nil {
			panic(err)
		}
		if tag.RowsAffected() == 0 {
			panic("MFA enrollment already exists; use an approved recovery process")
		}
		fmt.Println("Add this secret to an authenticator app, then run confirm with its current six-digit code:")
		fmt.Println(secret)
		fmt.Println("otpauth://totp/MatchMate:" + url.PathEscape(email) + "?secret=" + secret + "&issuer=MatchMate")
	case "confirm":
		if len(os.Args) != 4 {
			panic("usage: admin-mfa confirm EMAIL CODE")
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			panic(err)
		}
		defer tx.Rollback(ctx)
		var ciphertext []byte
		var confirmed *time.Time
		err = tx.QueryRow(ctx, `SELECT secret_ciphertext,confirmed_at FROM admin_totp WHERE account_id=$1 FOR UPDATE`, id).Scan(&ciphertext, &confirmed)
		if err != nil {
			panic(err)
		}
		if confirmed != nil {
			panic("MFA already confirmed")
		}
		secret, err := auth.DecryptMFASecret(key, ciphertext)
		if err != nil {
			panic(err)
		}
		step, err := auth.VerifyTOTP(secret, os.Args[3], time.Now().UTC(), -1)
		if err != nil {
			panic("invalid authenticator code")
		}
		if _, err = tx.Exec(ctx, `UPDATE admin_totp SET confirmed_at=now(),last_step=$2 WHERE account_id=$1`, id, step); err != nil {
			panic(err)
		}
		if _, err = tx.Exec(ctx, `UPDATE account SET token_version=token_version+1,updated_at=now() WHERE account_id=$1`, id); err != nil {
			panic(err)
		}
		if _, err = tx.Exec(ctx, `UPDATE refresh_session SET revoked_at=now() WHERE account_id=$1 AND revoked_at IS NULL`, id); err != nil {
			panic(err)
		}
		if err = tx.Commit(ctx); err != nil {
			panic(err)
		}
		fmt.Println("Administrator MFA confirmed. Sign in with password and a fresh code.")
	default:
		panic("usage: admin-mfa enroll EMAIL | confirm EMAIL CODE")
	}
}
