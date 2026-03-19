package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
)

func TestHealthRoutes(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 40023}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, target := range []string{"/", "/healthz", "/api/v1/healthz"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", target, rec.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("expected JSON payload for %s: %v", target, err)
		}

		if payload["status"] != "ok" {
			t.Fatalf("expected ok status for %s, got %q", target, payload["status"])
		}
	}
}
