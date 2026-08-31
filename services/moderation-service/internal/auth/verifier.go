package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
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
	expires                   time.Time
}
type jwks struct {
	Keys []struct {
		Kty, Use, Crv, Kid, X, Y string `json:"-"`
	} `json:"keys"`
}

func New(publicPEM, jwksURL, issuer, audience string) (*Verifier, error) {
	v := &Verifier{jwksURL: strings.TrimSpace(jwksURL), issuer: issuer, audience: audience, client: &http.Client{Timeout: 3 * time.Second}, keys: map[string]*ecdsa.PublicKey{}}
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
		return v.key(context.Background(), kid)
	}, jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !token.Valid {
		return domain.Principal{}, errors.New("invalid token")
	}
	if _, err = uuid.Parse(c.Subject); err != nil {
		return domain.Principal{}, errors.New("invalid token subject")
	}
	return domain.Principal{Subject: c.Subject, Roles: c.Roles}, nil
}

type jwkDocument struct {
	Keys []struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Crv string `json:"crv"`
		Kid string `json:"kid"`
		X   string `json:"x"`
		Y   string `json:"y"`
	} `json:"keys"`
}

func (v *Verifier) key(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Now().Before(v.expires)
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	res, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return nil, errors.New("JWK endpoint unavailable")
	}
	var doc jwkDocument
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&doc) != nil {
		return nil, errors.New("JWK response invalid")
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, item := range doc.Keys {
		if item.Kty != "EC" || item.Use != "sig" || item.Crv != "P-256" {
			continue
		}
		x, e1 := base64.RawURLEncoding.DecodeString(item.X)
		y, e2 := base64.RawURLEncoding.DecodeString(item.Y)
		if e1 != nil || e2 != nil {
			continue
		}
		px, py := elliptic.Unmarshal(elliptic.P256(), append([]byte{4}, append(x, y...)...))
		if px != nil {
			keys[item.Kid] = &ecdsa.PublicKey{Curve: elliptic.P256(), X: px, Y: py}
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
