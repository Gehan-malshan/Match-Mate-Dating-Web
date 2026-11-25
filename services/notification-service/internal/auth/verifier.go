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
	v := &Verifier{
		jwksURL:  strings.TrimSpace(jwksURL),
		issuer:   issuer,
		audience: audience,
		client:   &http.Client{Timeout: 3 * time.Second},
		keys:     map[string]*ecdsa.PublicKey{},
	}
	if strings.TrimSpace(publicPEM) != "" {
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(strings.ReplaceAll(publicPEM, `\n`, "\n")))
		if err != nil {
			return nil, err
		}
		v.key = key
	}
	if v.key == nil && v.jwksURL == "" {
		return nil, errors.New("NOTIFICATION_JWT_PUBLIC_KEY_PEM or ACCOUNT_JWKS_URL is required")
	}
	return v, nil
}

func (v *Verifier) Verify(raw string) (Principal, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodES256 {
			return nil, errors.New("unexpected signing method")
		}
		if v.key != nil {
			return v.key, nil
		}
		kid, _ := token.Header["kid"].(string)
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
	response, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("JWK endpoint unavailable")
	}
	var document jwksDocument
	if json.NewDecoder(response.Body).Decode(&document) != nil {
		return nil, errors.New("invalid JWK document")
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, item := range document.Keys {
		if item.KeyType != "EC" || item.Curve != "P-256" || item.Use != "sig" {
			continue
		}
		x, xErr := base64.RawURLEncoding.DecodeString(item.X)
		y, yErr := base64.RawURLEncoding.DecodeString(item.Y)
		if xErr != nil || yErr != nil {
			continue
		}
		pointX, pointY := elliptic.Unmarshal(elliptic.P256(), append([]byte{4}, append(x, y...)...))
		if pointX != nil {
			keys[item.KeyID] = &ecdsa.PublicKey{Curve: elliptic.P256(), X: pointX, Y: pointY}
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
