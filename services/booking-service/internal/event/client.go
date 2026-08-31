package event

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	template string
	http     *http.Client
}

func New(template string) *Client {
	return &Client{template: template, http: &http.Client{Timeout: 3 * time.Second}}
}
func (c *Client) Get(ctx context.Context, eventID string) (domain.EventSnapshot, error) {
	target := strings.Replace(c.template, "{eventId}", url.PathEscape(eventID), 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return domain.EventSnapshot{}, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return domain.EventSnapshot{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return domain.EventSnapshot{}, errors.New("event is unavailable")
	}
	var out domain.EventSnapshot
	err = json.NewDecoder(res.Body).Decode(&out)
	return out, err
}
