package recipient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

// Account resolves a recipient's email from the Account service over a
// service-authenticated internal endpoint. It never persists or logs the email.
type Account struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewAccount(baseURL, token string) *Account {
	return &Account{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: &http.Client{Timeout: 10 * time.Second}}
}

func (a *Account) Resolve(ctx context.Context, accountID string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/"+url.PathEscape(accountID), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("X-MatchMate-Internal-Token", a.token)
	response, err := a.client.Do(request)
	if err != nil {
		return "", &domain.SendFailure{Kind: domain.FailureRetryable, Code: "RECIPIENT_LOOKUP_UNAVAILABLE", Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusUnauthorized {
		return "", &domain.SendFailure{Kind: domain.FailurePermanent, Code: "RECIPIENT_UNAVAILABLE", Err: errors.New("recipient is unavailable")}
	}
	if response.StatusCode != http.StatusOK {
		return "", &domain.SendFailure{Kind: domain.FailureRetryable, Code: "RECIPIENT_LOOKUP_UNAVAILABLE", Err: errors.New("recipient lookup failed")}
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body); err != nil {
		return "", &domain.SendFailure{Kind: domain.FailureRetryable, Code: "RECIPIENT_LOOKUP_INVALID", Err: err}
	}
	parsed, err := mail.ParseAddress(body.Email)
	if err != nil || parsed.Address != body.Email {
		return "", &domain.SendFailure{Kind: domain.FailurePermanent, Code: "RECIPIENT_UNAVAILABLE", Err: errors.New("recipient email is invalid")}
	}
	return parsed.Address, nil
}
