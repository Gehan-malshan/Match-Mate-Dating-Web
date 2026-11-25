package securitymail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

func New(baseURL, internalToken string) func(context.Context, string, string, string) error {
	client := &http.Client{Timeout: 8 * time.Second}
	return func(ctx context.Context, email, purpose, token string) error {
		payload, err := json.Marshal(map[string]string{"email": email, "purpose": purpose, "token": token})
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/internal/security-email", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MatchMate-Internal-Token", internalToken)
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			return errors.New("security email delivery failed")
		}
		return nil
	}
}
