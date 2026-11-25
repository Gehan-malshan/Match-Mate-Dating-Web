package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Roles        []string `json:"roles"`
	TokenVersion int64    `json:"ver"`
	jwt.RegisteredClaims
}
type Manager struct {
	key                     *ecdsa.PrivateKey
	keyID, issuer, audience string
	ttl                     time.Duration
	previous                map[string]*ecdsa.PublicKey
}

func NewManager(privatePEM, keyID, issuer, audience string, ttl time.Duration) (*Manager, error) {
	var key *ecdsa.PrivateKey
	var err error
	if privatePEM == "" {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	} else {
		block, _ := pem.Decode([]byte(strings.ReplaceAll(privatePEM, `\n`, "\n")))
		if block == nil {
			return nil, errors.New("invalid JWT private key PEM")
		}
		parsed, e := x509.ParsePKCS8PrivateKey(block.Bytes)
		if e != nil {
			return nil, e
		}
		var ok bool
		key, ok = parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("JWT private key is not ECDSA")
		}
	}
	if err != nil {
		return nil, err
	}
	return &Manager{key: key, keyID: keyID, issuer: issuer, audience: audience, ttl: ttl, previous: map[string]*ecdsa.PublicKey{}}, nil
}

// SetPreviousKeys accepts a JSON map of old key IDs to public PEMs. Keep each
// public key published until every token signed by it has expired.
func (m *Manager) SetPreviousKeys(raw string) error {
	if raw == "" {
		return nil
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return err
	}
	for kid, pemValue := range values {
		if kid == "" || kid == m.keyID {
			return fmt.Errorf("invalid previous key id")
		}
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(pemValue))
		if err != nil {
			return err
		}
		if key.Curve != elliptic.P256() {
			return errors.New("previous key must use P-256")
		}
		m.previous[kid] = key
	}
	return nil
}
func (m *Manager) Issue(accountID string, roles []string, version int64, now time.Time) (string, time.Time, error) {
	exp := now.Add(m.ttl)
	claims := Claims{roles, version, jwt.RegisteredClaims{Issuer: m.issuer, Subject: accountID, Audience: jwt.ClaimStrings{m.audience}, ExpiresAt: jwt.NewNumericDate(exp), NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)), IssuedAt: jwt.NewNumericDate(now), ID: uuid.NewString()}}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = m.keyID
	signed, err := token.SignedString(m.key)
	return signed, exp, err
}
func (m *Manager) Parse(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodES256 {
			return nil, errors.New("unexpected signing method")
		}
		kid, _ := t.Header["kid"].(string)
		if kid == m.keyID {
			return &m.key.PublicKey, nil
		}
		if old, ok := m.previous[kid]; ok {
			return old, nil
		}
		return nil, errors.New("unknown signing key")
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience), jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !token.Valid {
		return nil, errors.New("invalid access token")
	}
	c, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return c, nil
}
func (m *Manager) JWK() map[string]any {
	keys := []map[string]any{jwk(m.keyID, &m.key.PublicKey)}
	for kid, key := range m.previous {
		keys = append(keys, jwk(kid, key))
	}
	return map[string]any{"keys": keys}
}
func jwk(kid string, key *ecdsa.PublicKey) map[string]any {
	x := base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32)))
	y := base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32)))
	return map[string]any{"kty": "EC", "use": "sig", "crv": "P-256", "alg": "ES256", "kid": kid, "x": x, "y": y}
}
func NewOpaqueToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	return raw, h[:], nil
}
func HashOpaqueToken(raw string) []byte { h := sha256.Sum256([]byte(raw)); return h[:] }
