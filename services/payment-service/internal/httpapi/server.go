package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gehan-malshan/matchmate/payment-service/internal/application"
	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
	"github.com/google/uuid"
)

type Verifier interface {
	Verify(string) (domain.Principal, error)
}
type Server struct {
	s   *application.Service
	v   Verifier
	log *slog.Logger
}

func New(s *application.Service, v Verifier, log *slog.Logger) http.Handler {
	h := &Server{s: s, v: v, log: log}
	m := http.NewServeMux()
	m.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	m.HandleFunc("POST /api/v1/payments/initiate", h.initiate)
	m.HandleFunc("GET /api/v1/bookings/{bookingId}/payment", h.get)
	m.HandleFunc("POST /api/v1/payments/payhere/callback", h.callback)
	return m
}
func (h *Server) principal(r *http.Request) (domain.Principal, string, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	p, e := h.v.Verify(raw)
	return p, raw, e
}
func (h *Server) initiate(w http.ResponseWriter, r *http.Request) {
	p, token, e := h.principal(r)
	if e != nil {
		problem(w, 401, "AUTHENTICATION_REQUIRED", "Authentication is required.")
		return
	}
	var in struct {
		BookingID string `json:"bookingId"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		City      string `json:"city"`
		Country   string `json:"country"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in) != nil {
		problem(w, 400, "INVALID_REQUEST", "Request body is invalid.")
		return
	}
	customer := application.CheckoutCustomer{FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, Phone: in.Phone, Address: in.Address, City: in.City, Country: in.Country}
	out, e := h.s.Initiate(r.Context(), p.Subject, token, in.BookingID, r.Header.Get("Idempotency-Key"), customer)
	if e != nil {
		problem(w, 409, "PAYMENT_INITIATION_REJECTED", "Payment cannot be initiated for this booking in its current state.")
		return
	}
	write(w, 201, out)
}
func (h *Server) get(w http.ResponseWriter, r *http.Request) {
	p, _, e := h.principal(r)
	if e != nil {
		problem(w, 401, "AUTHENTICATION_REQUIRED", "Authentication is required.")
		return
	}
	out, e := h.s.Get(r.Context(), p.Subject, r.PathValue("bookingId"))
	if e != nil {
		problem(w, 404, "PAYMENT_NOT_FOUND", "Payment was not found.")
		return
	}
	write(w, 200, out)
}
func (h *Server) callback(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if e := r.ParseForm(); e != nil {
		problem(w, 400, "INVALID_CALLBACK", "Callback is invalid.")
		return
	}
	n := payhere.Notification{MerchantID: r.FormValue("merchant_id"), OrderID: r.FormValue("order_id"), PaymentID: r.FormValue("payment_id"), Amount: r.FormValue("payhere_amount"), Currency: r.FormValue("payhere_currency"), StatusCode: r.FormValue("status_code"), Signature: r.FormValue("md5sig")}
	if e := h.s.Callback(r.Context(), n); e != nil {
		h.log.Error("payhere_callback_failed", "order_id", n.OrderID, "error", e)
		problem(w, 500, "CALLBACK_PROCESSING_FAILED", "Callback could not be processed.")
		return
	}
	w.WriteHeader(200)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://matchmate.example/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")), "title": detail, "status": status, "code": code, "detail": detail, "traceId": uuid.NewString()})
}
