package agentplatform

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestClaimsHelpers(t *testing.T) {
	t.Run("ticket agent claims", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindTicketAgent,
			PrincipalName: "builder",
			Scopes:        []string{string(ScopeTicketsCreate), string(ScopeProjectsUpdate)},
		}

		if !claims.HasScope(ScopeTicketsCreate) {
			t.Fatal("expected tickets.create scope to be present")
		}
		if claims.HasScope(ScopeTicketsList) {
			t.Fatal("did not expect tickets.list scope to be present")
		}
		if got := claims.CreatedBy(); got != "agent:builder" {
			t.Fatalf("CreatedBy() = %q", got)
		}
		if !claims.IsTicketAgent() {
			t.Fatal("expected IsTicketAgent() to be true")
		}
		if claims.IsProjectConversation() {
			t.Fatal("expected IsProjectConversation() to be false")
		}
	})

	t.Run("project conversation claims", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindProjectConversation,
			PrincipalName: "project-conversation:daily-sync",
		}
		if got := claims.CreatedBy(); got != "project-conversation:daily-sync" {
			t.Fatalf("CreatedBy() = %q", got)
		}
		if claims.IsTicketAgent() {
			t.Fatal("expected IsTicketAgent() to be false")
		}
		if !claims.IsProjectConversation() {
			t.Fatal("expected IsProjectConversation() to be true")
		}
	})

	t.Run("project conversation name gets prefix", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindProjectConversation,
			PrincipalName: "retro",
		}
		if got := claims.CreatedBy(); got != "project-conversation:retro" {
			t.Fatalf("CreatedBy() = %q", got)
		}
	})
}

func TestBuildEnvironment(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()

	got := BuildEnvironment(" https://openase.example/api ", " secret-token ", projectID, ticketID)
	want := []string{
		"OPENASE_PROJECT_ID=" + projectID.String(),
		"OPENASE_TICKET_ID=" + ticketID.String(),
		"OPENASE_API_URL=https://openase.example/api",
		"OPENASE_AGENT_TOKEN=secret-token",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildEnvironment() = %#v, want %#v", got, want)
	}

	withoutOptional := BuildEnvironment("   ", "   ", projectID, ticketID)
	if len(withoutOptional) != 2 {
		t.Fatalf("expected only required environment entries, got %#v", withoutOptional)
	}
}
