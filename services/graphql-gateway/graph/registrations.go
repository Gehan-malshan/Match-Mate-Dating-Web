package graph

import (
	"context"
	"errors"
	"net/http"

	"github.com/gehan-malshan/matchmate/graphql-gateway/graph/model"
)

func (r *queryResolver) registrationPage(ctx context.Context, path string, limit, offset *int) (*model.EventRegistrationPage, error) {
	if _, err := r.Upstream.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	pageSize, start := 25, 0
	if limit != nil {
		pageSize = *limit
	}
	if offset != nil {
		start = *offset
	}
	if pageSize < 1 || pageSize > 50 || start < 0 {
		return nil, errors.New("limit must be 1–50 and offset must be non-negative")
	}
	var bookings struct {
		Items []struct {
			BookingID     string  `json:"bookingId"`
			AccountID     string  `json:"accountId"`
			EventID       string  `json:"eventId"`
			State         string  `json:"state"`
			PaymentMethod string  `json:"paymentMethod"`
			Amount        string  `json:"amount"`
			Currency      string  `json:"currency"`
			CreatedAt     string  `json:"createdAt"`
			ConfirmedAt   *string `json:"confirmedAt"`
		} `json:"items"`
		HasMore bool `json:"hasMore"`
	}
	path += query(map[string]string{"limit": intValue(&pageSize, 25), "offset": intValue(&start, 0)})
	if err := r.Upstream.Do(ctx, r.Upstream.Services.Booking, path, http.MethodGet, nil, &bookings, nil); err != nil {
		return nil, err
	}
	items := make([]*model.EventRegistration, 0, len(bookings.Items))
	ids := make([]string, 0, len(bookings.Items))
	seen := map[string]bool{}
	for _, b := range bookings.Items {
		items = append(items, &model.EventRegistration{BookingID: b.BookingID, AccountID: b.AccountID, EventID: b.EventID, State: b.State, PaymentMethod: b.PaymentMethod, Amount: b.Amount, Currency: b.Currency, CreatedAt: b.CreatedAt, ConfirmedAt: b.ConfirmedAt})
		if !seen[b.AccountID] {
			ids = append(ids, b.AccountID)
			seen[b.AccountID] = true
		}
	}
	if len(ids) > 0 {
		var identities struct {
			Items []struct {
				AccountID string `json:"accountId"`
				Nickname  string `json:"nickname"`
				Email     string `json:"email"`
			} `json:"items"`
		}
		if err := r.Upstream.Do(ctx, r.Upstream.Services.Account, "/admin/member-identities", http.MethodPost, map[string]any{"accountIds": ids}, &identities, nil); err != nil {
			return nil, err
		}
		for _, identity := range identities.Items {
			for _, item := range items {
				if item.AccountID == identity.AccountID {
					item.Nickname = identity.Nickname
					item.Email = identity.Email
				}
			}
		}
	}
	return &model.EventRegistrationPage{Items: items, HasMore: bookings.HasMore}, nil
}
