package application

import (
	"context"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/google/uuid"
)

type feedRepositoryStub struct {
	records        []domain.FeedRecord
	hasMore        bool
	unread         int
	markAccount    string
	markID         string
	markAllAccount string
	markFound      bool
}

func (s *feedRepositoryStub) ListFeed(context.Context, string, int, *domain.FeedCursor) ([]domain.FeedRecord, bool, error) {
	return s.records, s.hasMore, nil
}
func (s *feedRepositoryStub) UnreadCount(context.Context, string) (int, error) { return s.unread, nil }
func (s *feedRepositoryStub) MarkRead(_ context.Context, accountID, notificationID string, _ time.Time) (bool, error) {
	s.markAccount, s.markID = accountID, notificationID
	return s.markFound, nil
}
func (s *feedRepositoryStub) MarkAllRead(_ context.Context, accountID string, _ time.Time) (int64, error) {
	s.markAllAccount = accountID
	return 3, nil
}

func TestFeedRendersSafeTemplateAndAction(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	id := uuid.NewString()
	repository := &feedRepositoryStub{
		unread: 1,
		records: []domain.FeedRecord{{
			ID: id, SourceEventType: "BookingConfirmed", Category: "BOOKING", CreatedAt: now,
			Template:  domain.Template{SubjectTemplate: "Booking confirmed", BodyTemplate: "Open MatchMate for details."},
			Variables: map[string]string{},
		}},
	}
	service := NewFeedService(repository)
	page, err := service.List(context.Background(), uuid.NewString(), 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ActionPath != "/app/bookings" || page.Items[0].Title != "Booking confirmed" || page.UnreadCount != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestFeedCursorRoundTripAndValidation(t *testing.T) {
	cursor := domain.FeedCursor{CreatedAt: time.Now().UTC().Round(0), ID: uuid.NewString()}
	decoded, err := decodeCursor(encodeCursor(cursor))
	if err != nil || !decoded.CreatedAt.Equal(cursor.CreatedAt) || decoded.ID != cursor.ID {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
	if _, err = decodeCursor("not-a-cursor"); err != ErrInvalidCursor {
		t.Fatalf("error=%v, want ErrInvalidCursor", err)
	}
}

func TestMarkReadConcealsOtherMembersNotifications(t *testing.T) {
	repository := &feedRepositoryStub{markFound: false}
	service := NewFeedService(repository)
	err := service.MarkRead(context.Background(), uuid.NewString(), uuid.NewString())
	if err != ErrNotificationMissing {
		t.Fatalf("error=%v, want ErrNotificationMissing", err)
	}
}
