package booking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
)

type Client struct {
	template string
	http     *http.Client
}

func New(template string) *Client {
	return &Client{template: template, http: &http.Client{Timeout: 3 * time.Second}}
}
func (c *Client) Snapshot(ctx context.Context, bookingID, token string) (domain.BookingSnapshot, error) {
	target := strings.Replace(c.template, "{bookingId}", url.PathEscape(bookingID), 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return domain.BookingSnapshot{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return domain.BookingSnapshot{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return domain.BookingSnapshot{}, errors.New("booking snapshot unavailable or ineligible")
	}
	var out domain.BookingSnapshot
	err = json.NewDecoder(res.Body).Decode(&out)
	return out, err
}
