package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func newTicketService(client *ent.Client) *ticketservice.Service {
	return ticketservice.NewService(ticketservice.Dependencies{
		Activity: ticketrepo.NewActivityRepository(client),
		Query:    ticketrepo.NewQueryRepository(client),
		Command:  ticketrepo.NewCommandRepository(client),
		Link:     ticketrepo.NewLinkRepository(client),
		Comment:  ticketrepo.NewCommentRepository(client),
		Usage:    ticketrepo.NewUsageRepository(client),
		Runtime:  ticketrepo.NewRuntimeRepository(client),
	})
}

func newInstanceAuthTestService(t *testing.T, bootstrap config.AuthConfig, configPath string) (*ent.Client, *accesscontrolservice.Service) {
	t.Helper()

	client := openTestEntClient(t)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	service, err := accesscontrolservice.New(
		accesscontrolrepo.NewEntRepository(client),
		t.Name()+":"+configPath,
		configPath,
		"",
	)
	if err != nil {
		t.Fatalf("new instance auth service: %v", err)
	}
	if bootstrap.Mode == config.AuthModeOIDC {
		now := time.Now().UTC()
		_, err = service.Activate(context.Background(), testActiveOIDCConfig(bootstrap), iam.OIDCActivationMetadata{
			ActivatedAt: &now,
			Source:      "test-bootstrap",
		})
		if err != nil {
			t.Fatalf("seed active instance auth state: %v", err)
		}
	}
	return client, service
}

func testActiveOIDCConfig(cfg config.AuthConfig) iam.ActiveOIDCConfig {
	claims := iam.DefaultDraftOIDCConfig().Claims
	if cfg.OIDC.EmailClaim != "" {
		claims.EmailClaim = cfg.OIDC.EmailClaim
	}
	if cfg.OIDC.NameClaim != "" {
		claims.NameClaim = cfg.OIDC.NameClaim
	}
	if cfg.OIDC.UsernameClaim != "" {
		claims.UsernameClaim = cfg.OIDC.UsernameClaim
	}
	if cfg.OIDC.GroupsClaim != "" {
		claims.GroupsClaim = cfg.OIDC.GroupsClaim
	}
	sessionPolicy := iam.DefaultDraftOIDCConfig().SessionPolicy
	if cfg.OIDC.SessionTTL > 0 {
		sessionPolicy.SessionTTL = cfg.OIDC.SessionTTL
	}
	if cfg.OIDC.SessionIdleTTL > 0 {
		sessionPolicy.SessionIdleTTL = cfg.OIDC.SessionIdleTTL
	}
	return iam.ActiveOIDCConfig{
		IssuerURL:            cfg.OIDC.IssuerURL,
		ClientID:             cfg.OIDC.ClientID,
		ClientSecret:         cfg.OIDC.ClientSecret,
		RedirectMode:         iam.OIDCRedirectModeFixed,
		FixedRedirectURL:     cfg.OIDC.RedirectURL,
		Scopes:               append([]string(nil), cfg.OIDC.Scopes...),
		Claims:               claims,
		AllowedEmailDomains:  append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
		SessionPolicy:        sessionPolicy,
	}
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

func executeJSONWithWriteActor(
	t *testing.T,
	server *Server,
	method string,
	target string,
	body any,
	writeActor string,
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

	rec := performJSONRequestWithWriteActor(t, server, method, target, bodyJSON, writeActor)
	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}
	if out != nil && wantStatus != http.StatusNoContent && rec.Body.Len() > 0 {
		decodeResponse(t, rec, out)
	}
}

func performJSONRequestWithWriteActor(
	t *testing.T,
	server *Server,
	method string,
	target string,
	body string,
	writeActor string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	if writeActor != "" {
		req = req.WithContext(withWriteActor(req.Context(), writeActor))
	}
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	return rec
}

func subscribeTopicEvents(
	t *testing.T,
	bus *eventinfra.ChannelBus,
	topic provider.Topic,
) <-chan provider.Event {
	t.Helper()

	streamCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stream, err := bus.Subscribe(streamCtx, topic)
	if err != nil {
		t.Fatalf("subscribe %s: %v", topic, err)
	}
	return stream
}

func readEvents(
	t *testing.T,
	stream <-chan provider.Event,
	count int,
) []provider.Event {
	t.Helper()

	events := make([]provider.Event, 0, count)
	for len(events) < count {
		select {
		case event := <-stream:
			events = append(events, event)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %d events, got %d", count, len(events))
		}
	}

	return events
}

func readEventType(
	t *testing.T,
	stream <-chan provider.Event,
	want provider.EventType,
	maxReads int,
) provider.Event {
	t.Helper()

	seen := make([]provider.EventType, 0, maxReads)
	for len(seen) < maxReads {
		select {
		case event := <-stream:
			seen = append(seen, event.Type)
			if event.Type == want {
				return event
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event %s after %d reads; saw %v", want, len(seen), seen)
		}
	}

	t.Fatalf("expected event %s within %d reads; saw %v", want, maxReads, seen)
	return provider.Event{}
}

func readTicketEventTicketIDs(
	t *testing.T,
	stream <-chan provider.Event,
	count int,
) []string {
	t.Helper()

	events := readEvents(t, stream, count)
	ids := make([]string, 0, len(events))
	for _, event := range events {
		var payload struct {
			Ticket struct {
				ID string `json:"id"`
			} `json:"ticket"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode ticket event payload: %v", err)
		}
		ids = append(ids, payload.Ticket.ID)
	}

	return ids
}

func assertStringSet(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected item count: got %v want %v", got, want)
	}
	sortedGot := append([]string(nil), got...)
	sortedWant := append([]string(nil), want...)
	slices.Sort(sortedGot)
	slices.Sort(sortedWant)
	if !slices.Equal(sortedGot, sortedWant) {
		t.Fatalf("unexpected items: got %s want %s", fmt.Sprint(sortedGot), fmt.Sprint(sortedWant))
	}
}
