package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func newTicketService(client *ent.Client) *ticketservice.Service {
	return ticketservice.NewService(ticketrepo.NewEntRepository(client))
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
