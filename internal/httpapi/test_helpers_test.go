package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func findStatusIDByName(t *testing.T, statuses []ticketstatus.Status, name string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}

	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.UUID{}
}

func executeJSON(
	t *testing.T,
	server *Server,
	method string,
	target string,
	body any,
	wantStatus int,
	out any,
) {
	t.Helper()

	bodyJSON := ""
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyJSON = string(encoded)
	}

	rec := performJSONRequest(t, server, method, target, bodyJSON)
	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}
	if out != nil && wantStatus != http.StatusNoContent && rec.Body.Len() > 0 {
		decodeResponse(t, rec, out)
	}
}
