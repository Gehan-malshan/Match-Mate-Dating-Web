package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/auth"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
)

const testEventID = "9fc263e0-3972-47c0-bf79-d9fa5e9d1201"

type testVerifier struct{ roles []string }
func (v testVerifier) Verify(string) (auth.Principal, error) { return auth.Principal{Subject:"admin-id", Roles:v.roles}, nil }

type testEvents struct{}
func (testEvents) Get(context.Context, string) (domain.EventSnapshot, error) { return domain.EventSnapshot{}, nil }

type testRepository struct { application.Repository; items []domain.Booking }
func (r testRepository) ListForEvent(context.Context, string, int, int) ([]domain.Booking, error) { return r.items, nil }

func TestEventRegistrationsRequireAdmin(t *testing.T) {
	for _, tc := range []struct { roles []string; status int }{{[]string{"member"}, 403}, {[]string{"admin"}, 200}} {
		repo := testRepository{items: []domain.Booking{{ID:"booking-id",AccountID:"member-id",EventID:testEventID,State:domain.Confirmed,PaymentMethod:"AT_VENUE",Amount:"2500.00",Currency:"LKR",CreatedAt:time.Now()}}}
		handler := New(application.New(repo, testEvents{}, 15*time.Minute), testVerifier{tc.roles}, slog.Default())
		request := httptest.NewRequest(http.MethodGet,"/api/v1/admin/events/"+testEventID+"/registrations?limit=25&offset=0",nil)
		request.Header.Set("Authorization", "Bearer test")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response,request)
		if response.Code != tc.status { t.Fatalf("roles %v: expected %d, got %d",tc.roles,tc.status,response.Code) }
		if tc.status == 200 {
			var body struct { Items []struct { AccountID string `json:"accountId"`; PaymentMethod string `json:"paymentMethod"` } `json:"items"` }
			if err := json.Unmarshal(response.Body.Bytes(),&body); err != nil { t.Fatal(err) }
			if len(body.Items) != 1 || body.Items[0].AccountID != "member-id" || body.Items[0].PaymentMethod != "AT_VENUE" { t.Fatalf("unexpected registrations: %+v",body) }
		}
	}
}
