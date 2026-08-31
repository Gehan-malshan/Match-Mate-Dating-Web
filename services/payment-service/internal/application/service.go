package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
	"github.com/google/uuid"
)

var ErrConflict = errors.New("conflict")
var ErrNotFound = errors.New("not found")

type BookingReader interface {
	Snapshot(context.Context, string, string) (domain.BookingSnapshot, error)
}
type Repository interface {
	Initiate(context.Context, domain.Payment, string, string) (domain.Payment, bool, error)
	FindByBooking(context.Context, string, string) (domain.Payment, error)
	ApplyCallback(context.Context, payhere.Notification, string, time.Time) (domain.Payment, bool, error)
}
type Clock func() time.Time
type Service struct {
	repo                            Repository
	bookings                        BookingReader
	provider                        payhere.Config
	now                             Clock
	returnURL, cancelURL, notifyURL string
}
type Checkout struct {
	Payment   domain.Payment    `json:"payment"`
	ActionURL string            `json:"actionUrl"`
	Fields    map[string]string `json:"fields"`
}
type CheckoutCustomer struct{ FirstName, LastName, Email, Phone, Address, City, Country string }

func New(repo Repository, bookings BookingReader, provider payhere.Config) *Service {
	return &Service{repo: repo, bookings: bookings, provider: provider, now: time.Now}
}
func (s *Service) WithCheckoutURLs(returnURL, cancelURL, notifyURL string) *Service {
	s.returnURL = returnURL
	s.cancelURL = cancelURL
	s.notifyURL = notifyURL
	return s
}

func (s *Service) Initiate(ctx context.Context, actor, token, bookingID, key string, customer CheckoutCustomer) (Checkout, error) {
	if bookingID == "" || key == "" {
		return Checkout{}, errors.New("bookingId and Idempotency-Key are required")
	}
	if customer.FirstName == "" || customer.LastName == "" || customer.Email == "" || customer.Phone == "" || customer.Address == "" || customer.City == "" || customer.Country == "" {
		return Checkout{}, errors.New("PayHere checkout contact fields are required")
	}
	snapshot, err := s.bookings.Snapshot(ctx, bookingID, token)
	if err != nil {
		return Checkout{}, err
	}
	now := s.now().UTC()
	if err = domain.ValidateSnapshot(snapshot, actor, now); err != nil {
		return Checkout{}, err
	}
	p := domain.Payment{ID: uuid.NewString(), BookingID: snapshot.BookingID, AccountID: snapshot.AccountID, OrderID: "MM-" + uuid.NewString(), Amount: snapshot.Amount, Currency: snapshot.Currency, Provider: "PAYHERE", State: domain.Pending, Version: 1, CreatedAt: now, UpdatedAt: now}
	fpBytes := sha256.Sum256([]byte(bookingID + "\x00" + customer.FirstName + "\x00" + customer.LastName + "\x00" + customer.Email + "\x00" + customer.Phone + "\x00" + customer.Address + "\x00" + customer.City + "\x00" + customer.Country))
	fp := hex.EncodeToString(fpBytes[:])
	p, _, err = s.repo.Initiate(ctx, p, key, fp)
	if err != nil {
		return Checkout{}, err
	}
	fields := map[string]string{"merchant_id": s.provider.MerchantID, "order_id": p.OrderID, "amount": p.Amount, "currency": p.Currency, "hash": payhere.RequestHash(s.provider.MerchantID, p.OrderID, p.Amount, p.Currency, s.provider.MerchantSecret), "return_url": s.returnURL, "cancel_url": s.cancelURL, "notify_url": s.notifyURL, "items": "MatchMate event booking", "first_name": customer.FirstName, "last_name": customer.LastName, "email": customer.Email, "phone": customer.Phone, "address": customer.Address, "city": customer.City, "country": customer.Country}
	return Checkout{Payment: p, ActionURL: s.provider.CheckoutURL, Fields: fields}, nil
}

func (s *Service) Get(ctx context.Context, actor, bookingID string) (domain.Payment, error) {
	return s.repo.FindByBooking(ctx, actor, bookingID)
}

func (s *Service) Callback(ctx context.Context, n payhere.Notification) error {
	now := s.now().UTC()
	verification := "VERIFIED"
	if err := payhere.VerifyNotification(n, s.provider); err != nil {
		verification = "REJECTED_" + err.Error()
	}
	_, _, err := s.repo.ApplyCallback(ctx, n, verification, now)
	return err
}
