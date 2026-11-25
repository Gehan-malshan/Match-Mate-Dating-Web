package application

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/gehan-malshan/matchmate/notification-service/internal/safety"
	"github.com/google/uuid"
)

var (
	ErrInvalidCursor       = errors.New("invalid notification cursor")
	ErrNotificationMissing = errors.New("notification not found")
)

type FeedRepository interface {
	ListFeed(context.Context, string, int, *domain.FeedCursor) ([]domain.FeedRecord, bool, error)
	UnreadCount(context.Context, string) (int, error)
	MarkRead(context.Context, string, string, time.Time) (bool, error)
	MarkAllRead(context.Context, string, time.Time) (int64, error)
}

type FeedPage struct {
	Items       []domain.FeedItem `json:"items"`
	NextCursor  string            `json:"nextCursor,omitempty"`
	UnreadCount int               `json:"unreadCount"`
}

type FeedService struct {
	repository FeedRepository
	now        func() time.Time
	safety     interface {
		Check(context.Context, string, string) (safety.Decision, error)
	}
}

func (s *FeedService) SetSafety(client interface {
	Check(context.Context, string, string) (safety.Decision, error)
}) {
	s.safety = client
}
func (s *FeedService) checkSafety(ctx context.Context, accountID, eventID string) (safety.Decision, error) {
	if s.safety == nil {
		return safety.Decision{}, nil
	}
	return s.safety.Check(ctx, accountID, eventID)
}

func NewFeedService(repository FeedRepository) *FeedService {
	return &FeedService{repository: repository, now: time.Now}
}

func (s *FeedService) List(ctx context.Context, accountID string, limit int, encodedCursor string) (FeedPage, error) {
	base, err := s.checkSafety(ctx, accountID, "")
	if err != nil {
		return FeedPage{}, err
	}
	if base.Suspended {
		return FeedPage{Items: []domain.FeedItem{}}, nil
	}
	if limit < 1 || limit > 50 {
		return FeedPage{}, ErrInvalidCursor
	}
	cursor, err := decodeCursor(encodedCursor)
	if err != nil {
		return FeedPage{}, err
	}
	records, hasMore, err := s.repository.ListFeed(ctx, accountID, limit, cursor)
	if err != nil {
		return FeedPage{}, err
	}
	items := make([]domain.FeedItem, 0, len(records))
	for _, record := range records {
		if s.safety != nil && record.Category == "BOOKING" {
			if record.EventID == "" {
				continue
			}
			decision, checkErr := s.checkSafety(ctx, accountID, record.EventID)
			if checkErr != nil {
				return FeedPage{}, checkErr
			}
			if decision.EventExcluded {
				continue
			}
		}
		rendered, renderErr := domain.Render(record.Template, record.Variables)
		if renderErr != nil {
			return FeedPage{}, fmt.Errorf("render member notification: %w", renderErr)
		}
		if len(rendered.Subject) > 160 || len(rendered.Body) > 2000 {
			return FeedPage{}, errors.New("rendered member notification exceeds contract limits")
		}
		items = append(items, domain.FeedItem{
			ID:              record.ID,
			SourceEventType: record.SourceEventType,
			Category:        record.Category,
			Title:           rendered.Subject,
			Message:         rendered.Body,
			ActionPath:      actionPath(record.SourceEventType),
			ReadAt:          record.ReadAt,
			CreatedAt:       record.CreatedAt,
		})
	}
	unread, err := s.UnreadCount(ctx, accountID)
	if err != nil {
		return FeedPage{}, err
	}
	page := FeedPage{Items: items, UnreadCount: unread}
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		page.NextCursor = encodeCursor(domain.FeedCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return page, nil
}

func (s *FeedService) UnreadCount(ctx context.Context, accountID string) (int, error) {
	decision, err := s.checkSafety(ctx, accountID, "")
	if err != nil {
		return 0, err
	}
	if decision.Suspended {
		return 0, nil
	}
	if s.safety != nil {
		grouped, ok := s.repository.(interface {
			UnreadGroups(context.Context, string) ([]domain.UnreadGroup, error)
		})
		if !ok {
			return 0, errors.New("notification safety unread projection unavailable")
		}
		groups, err := grouped.UnreadGroups(ctx, accountID)
		if err != nil {
			return 0, err
		}
		total := 0
		for _, group := range groups {
			if group.Category == "BOOKING" {
				if group.EventID == "" {
					continue
				}
				decision, checkErr := s.checkSafety(ctx, accountID, group.EventID)
				if checkErr != nil {
					return 0, checkErr
				}
				if decision.EventExcluded {
					continue
				}
			}
			total += group.Count
		}
		return total, nil
	}
	return s.repository.UnreadCount(ctx, accountID)
}

func (s *FeedService) MarkRead(ctx context.Context, accountID, notificationID string) error {
	decision, err := s.checkSafety(ctx, accountID, "")
	if err != nil {
		return err
	}
	if decision.Suspended {
		return ErrNotificationMissing
	}
	if _, err := uuid.Parse(notificationID); err != nil {
		return ErrNotificationMissing
	}
	updated, err := s.repository.MarkRead(ctx, accountID, notificationID, s.now().UTC())
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotificationMissing
	}
	return nil
}

func (s *FeedService) MarkAllRead(ctx context.Context, accountID string) (int64, error) {
	decision, err := s.checkSafety(ctx, accountID, "")
	if err != nil {
		return 0, err
	}
	if decision.Suspended {
		return 0, nil
	}
	return s.repository.MarkAllRead(ctx, accountID, s.now().UTC())
}

func actionPath(eventType string) string {
	if strings.HasPrefix(eventType, "Booking") || eventType == "HoldExpired" {
		return "/app/bookings"
	}
	return "/app/profile"
}

func encodeCursor(cursor domain.FeedCursor) string {
	value := strconv.FormatInt(cursor.CreatedAt.UTC().UnixNano(), 10) + "|" + cursor.ID
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeCursor(value string) (*domain.FeedCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return nil, ErrInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	if _, err = uuid.Parse(parts[1]); err != nil {
		return nil, ErrInvalidCursor
	}
	return &domain.FeedCursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: parts[1]}, nil
}
