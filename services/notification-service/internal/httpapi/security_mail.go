package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/gehan-malshan/matchmate/notification-service/internal/provider"
	"github.com/google/uuid"
)

// SecurityMail is a narrow, service-authenticated transient delivery path.
// One-time tokens never enter RabbitMQ, notification tables, or application logs.
func SecurityMail(next http.Handler, sender provider.Sender, internalToken, publicURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/internal/security-email", func(w http.ResponseWriter, r *http.Request) {
		if internalToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-MatchMate-Internal-Token")), []byte(internalToken)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var in struct {
			Email   string `json:"email"`
			Purpose string `json:"purpose"`
			Token   string `json:"token"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&in) != nil || len(in.Token) != 43 || strings.ContainsAny(in.Token, "\r\n") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		address, err := mail.ParseAddress(in.Email)
		if err != nil || address.Address != in.Email {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		path, subject, text := "", "", ""
		switch in.Purpose {
		case "verify":
			path, subject, text = "/verify-email", "Verify your MatchMate email", "Verify your MatchMate email using this link: "
		case "reset":
			path, subject, text = "/reset-password", "Reset your MatchMate password", "Reset your MatchMate password using this link: "
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		link := strings.TrimRight(publicURL, "/") + path + "?token=" + url.QueryEscape(in.Token)
		_, err = sender.Send(r.Context(), domain.Delivery{ID: uuid.NewString()}, domain.RenderedMessage{Subject: subject, Body: text + link + "\nIf you did not request this, ignore this message."}, in.Email)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", next)
	return mux
}
