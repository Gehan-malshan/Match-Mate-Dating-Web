package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
)

func TestEventImageProxy(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events/"+id+"/image" {
			t.Errorf("unexpected upstream path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg-bytes"))
	}))
	defer provider.Close()
	client := &upstream.Client{HTTP: provider.Client(), Services: upstream.Services{Event: provider.URL + "/api/v1"}}
	handler := eventImageProxy(client)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/media/events/"+id, nil)
	r.SetPathValue("eventId", id)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Body.String() != "jpeg-bytes" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("proxy response: %d %q", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/media/events/bad", nil)
	r.SetPathValue("eventId", "bad")
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("invalid id accepted: %d", w.Code)
	}
}
