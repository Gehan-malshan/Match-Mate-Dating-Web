package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}
type Verifier struct {
	staticKey                 *ecdsa.PublicKey
	jwksURL, issuer, audience string
	client                    *http.Client
	mu                        sync.RWMutex
	keys                      map[string]*ecdsa.PublicKey
	keysExpireAt              time.Time
	now                       func() time.Time
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
	v := &Verifier{jwksURL: strings.TrimSpace(jwksURL), issuer: issuer, audience: audience, client: &http.Client{Timeout: 3 * time.Second}, keys: map[string]*ecdsa.PublicKey{}, now: time.Now}
	if strings.TrimSpace(publicPEM) != "" {
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(strings.ReplaceAll(publicPEM, `\n`, "\n")))
		if err != nil {
			return nil, err
		}
		v.staticKey = key
	}
	if v.staticKey == nil && v.jwksURL == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PEM or ACCOUNT_JWKS_URL is required")
	}
	return v, nil
}
func (v *Verifier) Verify(raw string) (domain.Principal, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodES256 {
			return nil, errors.New("unexpected signing method")
		}
		if v.staticKey != nil {
			return v.staticKey, nil
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing key id")
		}
		return v.key(context.Background(), kid)
	}, jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !token.Valid || c.Subject == "" {
		return domain.Principal{}, errors.New("invalid token")
	}
	return domain.Principal{Subject: c.Subject, Roles: c.Roles}, nil
}
func (v *Verifier) key(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := v.now().Before(v.keysExpireAt)
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, errors.New("signing key not found")
	}
	return key, nil
}
func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return errors.New("JWK endpoint unavailable")
	}
	var doc jwksDocument
	if err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, item := range doc.Keys {
		if item.KeyType != "EC" || item.Curve != "P-256" || item.Use != "sig" || item.KeyID == "" {
			continue
		}
		x, xErr := base64.RawURLEncoding.DecodeString(item.X)
		y, yErr := base64.RawURLEncoding.DecodeString(item.Y)
		if xErr != nil || yErr != nil {
			continue
		}
		pointX, pointY := elliptic.Unmarshal(elliptic.P256(), append([]byte{4}, append(x, y...)...))
		if pointX == nil || pointY == nil {
			continue
		}
		keys[item.KeyID] = &ecdsa.PublicKey{Curve: elliptic.P256(), X: pointX, Y: pointY}
	}
	if len(keys) == 0 {
		return errors.New("JWK endpoint returned no supported signing keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.keysExpireAt = v.now().Add(5 * time.Minute)
	v.mu.Unlock()
	return nil
}
