package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/graphql-gateway/graph/model"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/transport"
)

type Services struct {
	Account, Event, Matchmaking, Booking, Payment, Notification string
}

type Client struct {
	HTTP     *http.Client
	Services Services
}

type Problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Status int    `json:"-"`
}

func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Detail
	}
	return fmt.Sprintf("upstream request failed with status %d", p.Status)
}

func New(services Services) *Client {
	return &Client{HTTP: &http.Client{Timeout: 10 * time.Second}, Services: services}
}

func Query(values map[string]string) string {
	query := url.Values{}
	for key, value := range values {
		if value != "" {
			query.Set(key, value)
		}
	}
	if len(query) == 0 {
		return ""
	}
	return "?" + query.Encode()
}

func (c *Client) Do(ctx context.Context, base, path, method string, body any, target any, headers map[string]string) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if incoming := transport.Request(ctx); incoming != nil {
		if authorization := incoming.Header.Get("Authorization"); authorization != "" {
			request.Header.Set("Authorization", authorization)
		}
		if cookie := incoming.Header.Get("Cookie"); cookie != "" {
			request.Header.Set("Cookie", cookie)
		}
		if origin := incoming.Header.Get("Origin"); origin != "" {
			request.Header.Set("Origin", origin)
		}
		if correlation := incoming.Header.Get("X-Correlation-ID"); correlation != "" {
			request.Header.Set("X-Correlation-ID", correlation)
		}
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("service unavailable: %w", err)
	}
	defer response.Body.Close()
	if writer := transport.Writer(ctx); writer != nil {
		for _, cookie := range response.Header.Values("Set-Cookie") {
			writer.Header().Add("Set-Cookie", strings.Replace(cookie, "Path=/api/v1/auth", "Path=/", 1))
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		problem := &Problem{Status: response.StatusCode, Code: "UPSTREAM_ERROR"}
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(problem)
		return problem
	}
	if target == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(target); err != nil {
		return fmt.Errorf("invalid service response: %w", err)
	}
	return nil
}

func (c *Client) RequireAdmin(ctx context.Context) (*model.Account, error) {
	var result struct {
		Account *model.Account `json:"account"`
	}
	if err := c.Do(ctx, c.Services.Account, "/users/me", http.MethodGet, nil, &result, nil); err != nil {
		return nil, err
	}
	if result.Account == nil {
		return nil, errors.New("account response is missing")
	}
	for _, role := range result.Account.Roles {
		if strings.EqualFold(role, "admin") {
			return result.Account, nil
		}
	}
	return nil, &Problem{Code: "ADMIN_ROLE_REQUIRED", Detail: "Administrator access is required.", Status: http.StatusForbidden}
}
