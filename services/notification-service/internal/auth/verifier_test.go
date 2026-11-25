package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifierAcceptsAccountES256TokenAndRejectsWrongAudience(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	verifier, err := New(string(publicPEM), "", "matchmate-account", "matchmate-api")
	if err != nil {
		t.Fatal(err)
	}
	makeToken := func(audience string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims{
			Roles: []string{"member"},
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "account-id",
				Issuer:    "matchmate-account",
				Audience:  jwt.ClaimStrings{audience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			},
		})
		signed, signErr := token.SignedString(privateKey)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return signed
	}
	principal, err := verifier.Verify(makeToken("matchmate-api"))
	if err != nil || principal.Subject != "account-id" || len(principal.Roles) != 1 {
		t.Fatalf("principal=%+v err=%v", principal, err)
	}
	if _, err = verifier.Verify(makeToken("another-api")); err == nil {
		t.Fatal("wrong audience must be rejected")
	}
}
