package auth

import (
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("a correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("a correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("password should verify: %v", err)
	}
	ok, _ = VerifyPassword("wrong password", hash)
	if ok {
		t.Fatal("wrong password verified")
	}
}
func TestAccessTokenAndJWK(t *testing.T) {
	m, err := NewManager("", "test-key", "issuer", "audience", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := m.Issue("account-1", []string{"member"}, 3, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "account-1" || claims.TokenVersion != 3 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(m.JWK()["keys"].([]map[string]any)) != 1 {
		t.Fatal("missing public JWK")
	}
}
func TestOpaqueTokensAreHashed(t *testing.T) {
	raw, hash, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || string(hash) == raw {
		t.Fatal("opaque token was not separated from its hash")
	}
	again := HashOpaqueToken(raw)
	if string(hash) != string(again) {
		t.Fatal("token hash is not stable")
	}
}
