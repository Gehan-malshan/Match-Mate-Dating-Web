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
	"github.com/google/uuid"
)

func testVerifier(t *testing.T) (*Verifier, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	verifier, err := New(string(publicPEM), "", "matchmate-account", "matchmate-api")
	if err != nil {
		t.Fatal(err)
	}
	return verifier, key
}

func signedToken(t *testing.T, key *ecdsa.PrivateKey, subject, issuer, audience string, expires time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims{
		Roles: []string{"member"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestVerifierAcceptsValidES256Token(t *testing.T) {
	verifier, key := testVerifier(t)
	subject := uuid.NewString()
	principal, err := verifier.Verify(signedToken(t, key, subject, "matchmate-account", "matchmate-api", time.Now().Add(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	if principal.Subject != subject || !principal.HasRole("member") {
		t.Fatalf("principal=%+v", principal)
	}
}

func TestVerifierRejectsInvalidClaims(t *testing.T) {
	verifier, key := testVerifier(t)
	tests := map[string]string{
		"subject":  signedToken(t, key, "not-a-uuid", "matchmate-account", "matchmate-api", time.Now().Add(time.Minute)),
		"issuer":   signedToken(t, key, uuid.NewString(), "another-issuer", "matchmate-api", time.Now().Add(time.Minute)),
		"audience": signedToken(t, key, uuid.NewString(), "matchmate-account", "another-api", time.Now().Add(time.Minute)),
		"expired":  signedToken(t, key, uuid.NewString(), "matchmate-account", "matchmate-api", time.Now().Add(-time.Minute)),
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Verify(raw); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
}

func TestVerifierRejectsUnexpectedSigningAlgorithm(t *testing.T) {
	verifier, _ := testVerifier(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.NewString(), "iss": "matchmate-account", "aud": "matchmate-api", "exp": time.Now().Add(time.Minute).Unix(),
	})
	raw, err := token.SignedString([]byte("not-an-ecdsa-key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = verifier.Verify(raw); err == nil {
		t.Fatal("unexpected signing algorithm accepted")
	}
}
