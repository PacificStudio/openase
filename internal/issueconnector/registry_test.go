package issueconnector

import (
	"context"
	"net/http"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	connector := stubConnector{id: "github"}
	if err := registry.Register(connector); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if err := registry.Register(connector); err == nil {
		t.Fatalf("expected duplicate registration error")
	}

	got, err := registry.Get(domain.TypeGitHub)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Name() != "github" {
		t.Fatalf("Name = %q, want %q", got.Name(), "github")
	}
}

func TestRegistryListTypesIsSorted(t *testing.T) {
	registry, err := NewRegistry(
		stubConnector{id: "gitlab", name: "GitLab"},
		stubConnector{id: "github", name: "GitHub"},
	)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	got := registry.ListTypes()
	want := []domain.Type{domain.TypeGitHub, domain.TypeGitLab}
	if len(got) != len(want) {
		t.Fatalf("ListTypes length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListTypes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRegistryListTypesReturnsEmptySliceWhenRegistryIsEmpty(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	got := registry.ListTypes()
	if got == nil {
		t.Fatalf("ListTypes returned nil")
	}
	if len(got) != 0 {
		t.Fatalf("ListTypes length = %d, want 0", len(got))
	}
}

func TestRegistryCoversNilAndInvalidBranches(t *testing.T) {
	var nilRegistry *Registry
	if err := nilRegistry.Register(stubConnector{id: "github"}); err == nil || err.Error() != "connector registry is nil" {
		t.Fatalf("expected nil registry register error, got %v", err)
	}
	if _, err := nilRegistry.Get(domain.TypeGitHub); err == nil || err.Error() != "connector registry is nil" {
		t.Fatalf("expected nil registry get error, got %v", err)
	}
	if got := nilRegistry.ListTypes(); got != nil {
		t.Fatalf("expected nil ListTypes on nil registry, got %#v", got)
	}

	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}
	if err := registry.Register(nil); err == nil || err.Error() != "connector must not be nil" {
		t.Fatalf("expected nil connector error, got %v", err)
	}
	if err := registry.Register(stubConnector{id: "   "}); err == nil || err.Error() == "" {
		t.Fatalf("expected invalid connector id error, got %v", err)
	}
	if _, err := registry.Get(domain.TypeGitLab); err == nil || err.Error() != `connector "gitlab" is not registered` {
		t.Fatalf("expected missing connector error, got %v", err)
	}

	if _, err := NewRegistry(stubConnector{id: "   "}); err == nil || err.Error() == "" {
		t.Fatalf("expected constructor validation error, got %v", err)
	}
}

type stubConnector struct {
	id   string
	name string
}

func (s stubConnector) ID() string {
	return s.id
}

func (s stubConnector) Name() string {
	if s.name != "" {
		return s.name
	}
	return s.id
}

func (s stubConnector) PullIssues(context.Context, domain.Config, time.Time) ([]domain.ExternalIssue, error) {
	return nil, nil
}

func (s stubConnector) ParseWebhook(context.Context, http.Header, []byte) (*domain.WebhookEvent, error) {
	return nil, nil
}

func (s stubConnector) SyncBack(context.Context, domain.Config, domain.SyncBackUpdate) error {
	return nil
}

func (s stubConnector) HealthCheck(context.Context, domain.Config) error {
	return nil
}
