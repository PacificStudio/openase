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
