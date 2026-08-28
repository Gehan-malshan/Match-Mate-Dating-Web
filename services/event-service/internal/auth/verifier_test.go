package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifierLoadsAccountJWKAndValidatesClaims(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "EC", "use": "sig", "crv": "P-256", "alg": "ES256", "kid": "test-key", "x": base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32)))}}})
	}))
	defer server.Close()
	v, err := New("", server.URL, "issuer", "audience")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims{Roles: []string{"organizer"}, RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Subject: "organizer-1", Audience: jwt.ClaimStrings{"audience"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}})
	token.Header["kid"] = "test-key"
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := v.Verify(raw)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Subject != "organizer-1" || !principal.HasRole("organizer") {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if _, err = v.Verify(raw); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("expected cached JWK, got %d requests", requests)
	}
}

func TestVerifierRejectsWrongAudience(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "EC", "use": "sig", "crv": "P-256", "kid": "k", "x": base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32)))}}})
	}))
	defer server.Close()
	v, _ := New("", server.URL, "issuer", "expected")
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Subject: "member", Audience: jwt.ClaimStrings{"wrong"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}})
	token.Header["kid"] = "k"
	raw, _ := token.SignedString(key)
	if _, err := v.Verify(raw); err == nil {
		t.Fatal("expected wrong audience rejection")
	}
}
