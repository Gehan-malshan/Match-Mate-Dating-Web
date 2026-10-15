package graph

import (
	"context"
	"net/http"

	"github.com/gehan-malshan/matchmate/graphql-gateway/graph/model"
)

func (r *mutationResolver) setEventArchived(ctx context.Context, eventID string, expectedVersion int, action string) (*model.Event, error) {
	if _, err := r.Upstream.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	var result model.Event
	err := r.Upstream.Do(ctx, r.Upstream.Services.Event, "/events/"+escape(eventID)+"/"+action, http.MethodPost, map[string]any{"expectedVersion": expectedVersion}, &result, nil)
	return &result, err
}
