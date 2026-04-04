package ticket

import (
	"context"
	"errors"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestTicketServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.Background()
	projectID := uuid.New()
	ticketID := uuid.New()
	commentID := uuid.New()
	dependencyID := uuid.New()
	externalLinkID := uuid.New()

	if _, err := service.List(ctx, ListInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Get(ctx, ticketID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Create(ctx, CreateInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Update(ctx, UpdateInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Update error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddDependency error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveDependency(ctx, ticketID, dependencyID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveDependency error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddExternalLink error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListComments(ctx, ticketID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListComments error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddComment(ctx, AddCommentInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddComment error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateComment(ctx, UpdateCommentInput{TicketID: ticketID, CommentID: commentID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateComment error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListCommentRevisions(ctx, ticketID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListCommentRevisions error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveExternalLink(ctx, ticketID, externalLinkID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveExternalLink error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveComment(ctx, ticketID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveComment error = %v, want %v", err, ErrUnavailable)
	}
}

func TestTicketHelperFunctions(t *testing.T) {
	t.Parallel()

	statusID := uuid.New()

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v", got)
	}
	if value, ok := parseIdentifierSequence("ASE-42"); !ok || value != 42 {
		t.Fatalf("parseIdentifierSequence() = %d, %v", value, ok)
	}
	if _, ok := parseIdentifierSequence("OTHER-42"); ok {
		t.Fatal("parseIdentifierSequence() expected false for foreign prefix")
	}
	if got := resolveCreatedBy("  "); got != defaultCreatedBy {
		t.Fatalf("resolveCreatedBy(blank) = %q", got)
	}
	if got := resolveCreatedBy(" user:codex "); got != "user:codex" {
		t.Fatalf("resolveCreatedBy(value) = %q", got)
	}
	if !optionalUUIDPointerEqual(nil, nil) || optionalUUIDPointerEqual(&statusID, nil) || !optionalUUIDPointerEqual(&statusID, &statusID) {
		t.Fatal("optionalUUIDPointerEqual() returned unexpected results")
	}
}

func TestTicketRepoScopeAndMetricHelpers(t *testing.T) {
	t.Parallel()

	if got := timeNowUTC(); got.Location() != time.UTC {
		t.Fatalf("timeNowUTC() = %+v", got)
	}

	metrics := &ticketMetricsProvider{}
	agentItem := UsageMetricsAgent{
		ProviderName: "Codex",
		ModelName:    "gpt-5.4",
	}
	recordTokenUsageMetrics(metrics, agentItem, ticketing.UsageDelta{InputTokens: 10, OutputTokens: 5})
	recordCostUsageMetrics(metrics, agentItem, uuid.New(), 3.5)
	if len(metrics.calls) != 3 {
		t.Fatalf("metric calls = %+v", metrics.calls)
	}
	if got := mergeTags(provider.Tags{"provider": "Codex"}, provider.Tags{"direction": "input"}); got["provider"] != "Codex" || got["direction"] != "input" {
		t.Fatalf("mergeTags() = %+v", got)
	}

	recordTokenUsageMetrics(nil, agentItem, ticketing.UsageDelta{InputTokens: 1})
	recordTokenUsageMetrics(metrics, UsageMetricsAgent{}, ticketing.UsageDelta{InputTokens: 1})
	recordCostUsageMetrics(metrics, UsageMetricsAgent{}, uuid.New(), 1)
	recordCostUsageMetrics(metrics, agentItem, uuid.New(), 0)
}

type ticketMetricsProvider struct {
	calls []ticketMetricCall
}

type ticketMetricCall struct {
	name  string
	tags  provider.Tags
	value float64
}

func (m *ticketMetricsProvider) Counter(name string, tags provider.Tags) provider.Counter {
	return ticketCounterRecorder{
		provider: m,
		name:     name,
		tags:     tags,
	}
}

func (m *ticketMetricsProvider) Histogram(string, provider.Tags) provider.Histogram {
	return ticketHistogramRecorder{}
}

func (m *ticketMetricsProvider) Gauge(string, provider.Tags) provider.Gauge {
	return ticketGaugeRecorder{}
}

type ticketCounterRecorder struct {
	provider *ticketMetricsProvider
	name     string
	tags     provider.Tags
}

func (r ticketCounterRecorder) Add(value float64) {
	r.provider.calls = append(r.provider.calls, ticketMetricCall{
		name:  r.name,
		tags:  r.tags,
		value: value,
	})
}

type ticketHistogramRecorder struct{}

func (ticketHistogramRecorder) Record(float64) {}

type ticketGaugeRecorder struct{}

func (ticketGaugeRecorder) Set(float64) {}

var _ = entagentprovider.FieldID
var _ = entagentrun.FieldID
