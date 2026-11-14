package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strconv"
	"testing"
	"time"
)

func TestTOTPVectorAndReplay(t *testing.T) {
	// RFC 6238 SHA-1 test vector at T=59, truncated to six digits.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	step, err := VerifyTOTP(secret, "287082", time.Unix(59, 0), -1)
	if err != nil || step != 1 {
		t.Fatalf("vector step=%d err=%v", step, err)
	}
	if _, err = VerifyTOTP(secret, "287082", time.Unix(59, 0), step); err == nil {
		t.Fatal("replayed code accepted")
	}
	if _, err = VerifyTOTP(secret, "28708x", time.Unix(59, 0), -1); err == nil {
		t.Fatal("non-numeric code accepted")
	}
}

func TestMFASecretEncryption(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := EncryptMFASecret(key, "secret")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := DecryptMFASecret(key, ciphertext)
	if err != nil || plain != "secret" {
		t.Fatalf("decryption failed: %v", err)
	}
	wrong := make([]byte, 32)
	if _, err = DecryptMFASecret(wrong, ciphertext); err == nil {
		t.Fatal("wrong key accepted")
	}
}

func TestJWTOverlappingKeyRotation(t *testing.T) {
	old, err := NewManager("", "old", "issuer", "audience", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := old.Issue("account-1", []string{"member"}, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(newKey)
	if err != nil {
		t.Fatal(err)
	}
	newManager, err := NewManager(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), "new", "issuer", "audience", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&old.key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	previousJSON := `{"old":` + strconv.Quote(string(publicPEM)) + `}`
	if err = newManager.SetPreviousKeys(previousJSON); err != nil {
		t.Fatal(err)
	}
	if _, err = newManager.Parse(raw); err != nil {
		t.Fatalf("old token invalid during overlap: %v", err)
	}
	if len(newManager.JWK()["keys"].([]map[string]any)) != 2 {
		t.Fatal("old key missing from JWKS")
	}
}
