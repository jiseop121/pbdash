package pocketbase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientAuthenticateFallsBackToLegacyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/collections/_superusers/auth-with-password":
			w.WriteHeader(http.StatusNotFound)
		case "/api/admins/auth-with-password":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "tok-legacy"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	token, err := c.Authenticate(context.Background(), server.URL, "root@example.com", "pw")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if token != "tok-legacy" {
		t.Fatalf("token mismatch: got=%q", token)
	}
}

func TestClientGetJSONSupportsArrayPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "r1"}, {"id": "r2"}})
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	payload, err := c.GetJSON(context.Background(), server.URL, "tok", "/api/collections/posts/records", nil)
	if err != nil {
		t.Fatalf("get json: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok {
		t.Fatalf("expected items array payload: %#v", payload)
	}
	if len(items) != 2 {
		t.Fatalf("items length mismatch: got=%d want=2", len(items))
	}
}

func TestClientGetJSONReturnsAuthErrorOnUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := &Client{httpClient: server.Client()}
	_, err := c.GetJSON(context.Background(), server.URL, "tok", "/api/collections/posts", nil)
	if err == nil {
		t.Fatalf("expected auth error")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got=%T (%v)", err, err)
	}
}
