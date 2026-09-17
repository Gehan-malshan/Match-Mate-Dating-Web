package graph

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gehan-malshan/matchmate/graphql-gateway/graph/model"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
)

func (r *mutationResolver) adminMatchingCommand(ctx context.Context, path string, body any, key string) (*model.MatchingRun, error) {
	if _, err := r.Upstream.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	headers := map[string]string{}
	if key != "" {
		headers["Idempotency-Key"] = key
	}
	var result model.MatchingRun
	err := r.Upstream.Do(ctx, r.Upstream.Services.Matchmaking, path, http.MethodPost, body, &result, headers)
	return &result, err
}

func escape(value string) string {
	return url.PathEscape(value)
}
func query(values map[string]string) string {
	return upstream.Query(values)
}
func intValue(value *int, fallback int) string {
	if value == nil {
		return strconv.Itoa(fallback)
	}
	return strconv.Itoa(*value)
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
