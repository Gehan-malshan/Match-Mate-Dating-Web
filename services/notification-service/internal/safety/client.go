package safety

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Decision is the restricted, minimum-safe current moderation state.
type Decision struct {
	Suspended           bool `json:"suspended"`
	ProfileHidden       bool `json:"profileHidden"`
	EventExcluded       bool `json:"eventExcluded"`
	MatchmakingExcluded bool `json:"matchmakingExcluded"`
	RevealPrevented     bool `json:"revealPrevented"`
}

type Client struct {
	baseURL, token string
	http           *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: 2 * time.Second}}
}

func (c *Client) Check(ctx context.Context, accountID, eventID string) (Decision, error) {
	if c == nil || c.baseURL == "" || c.token == "" {
		return Decision{}, errors.New("moderation safety dependency is not configured")
	}
	endpoint := c.baseURL + "/api/v1/internal/safety/" + url.PathEscape(accountID)
	if eventID != "" {
		endpoint += "?eventId=" + url.QueryEscape(eventID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("X-Moderation-Service-Token", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return Decision{}, fmt.Errorf("moderation safety lookup: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Decision{}, fmt.Errorf("moderation safety lookup returned %d", resp.StatusCode)
	}
	var decision Decision
	raw := map[string]json.RawMessage{}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&raw); err != nil {
		return Decision{}, err
	}
	for _, field := range []string{"suspended", "profileHidden", "eventExcluded", "matchmakingExcluded", "revealPrevented"} {
		value, ok := raw[field]
		if !ok || (string(value) != "true" && string(value) != "false") {
			return Decision{}, fmt.Errorf("moderation safety response missing %s", field)
		}
	}
	encoded, _ := json.Marshal(raw)
	if err = json.Unmarshal(encoded, &decision); err != nil {
		return Decision{}, err
	}
	return decision, nil
}
