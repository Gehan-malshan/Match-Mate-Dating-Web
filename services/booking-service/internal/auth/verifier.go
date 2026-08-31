package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Principal struct {
	Subject string
	Roles   []string
}
type claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}
type Verifier struct {
	key              *ecdsa.PublicKey
	issuer, audience string
	jwksURL          string
	client           *http.Client
	mu               sync.RWMutex
	keys             map[string]*ecdsa.PublicKey
	expires          time.Time
}
type jwksDocument struct {
	Keys []struct {
		KeyType string `json:"kty"`
		Use     string `json:"use"`
		Curve   string `json:"crv"`
		KeyID   string `json:"kid"`
		X       string `json:"x"`
		Y       string `json:"y"`
	} `json:"keys"`
}

func New(publicPEM, jwksURL, issuer, audience string) (*Verifier, error) {
	v := &Verifier{jwksURL: strings.TrimSpace(jwksURL), issuer: issuer, audience: audience, client: &http.Client{Timeout: 3 * time.Second}, keys: map[string]*ecdsa.PublicKey{}}
	if strings.TrimSpace(publicPEM) != "" {
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(strings.ReplaceAll(publicPEM, `\n`, "\n")))
		if err != nil {
			return nil, err
		}
		v.key = key
	}
	if v.key == nil && v.jwksURL == "" {
		return nil, errors.New("BOOKING_JWT_PUBLIC_KEY_PEM or ACCOUNT_JWKS_URL is required")
	}
	return v, nil
}
func (v *Verifier) Verify(raw string) (Principal, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodES256 {
			return nil, errors.New("unexpected signing method")
		}
		if v.key != nil {
			return v.key, nil
		}
		kid, _ := t.Header["kid"].(string)
		return v.remoteKey(context.Background(), kid)
	}, jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !token.Valid || c.Subject == "" {
		return Principal{}, errors.New("invalid token")
	}
	return Principal{Subject: c.Subject, Roles: c.Roles}, nil
}

func (v *Verifier) remoteKey(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Now().Before(v.expires)
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("JWK endpoint unavailable")
	}
	var doc jwksDocument
	if json.NewDecoder(res.Body).Decode(&doc) != nil {
		return nil, errors.New("invalid JWK document")
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, item := range doc.Keys {
		if item.KeyType != "EC" || item.Curve != "P-256" || item.Use != "sig" {
			continue
		}
		x, xe := base64.RawURLEncoding.DecodeString(item.X)
		y, ye := base64.RawURLEncoding.DecodeString(item.Y)
		if xe != nil || ye != nil {
			continue
		}
		px, py := elliptic.Unmarshal(elliptic.P256(), append([]byte{4}, append(x, y...)...))
		if px != nil {
			keys[item.KeyID] = &ecdsa.PublicKey{Curve: elliptic.P256(), X: px, Y: py}
		}
	}
	v.mu.Lock()
	v.keys = keys
	v.expires = time.Now().Add(5 * time.Minute)
	key, ok = keys[kid]
	v.mu.Unlock()
	if !ok {
		return nil, errors.New("signing key not found")
	}
	return key, nil
}
