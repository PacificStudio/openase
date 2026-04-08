package secrets

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseScopeKind(t *testing.T) {
	tests := []struct {
		input string
		want  ScopeKind
		ok    bool
	}{
		{input: "organization", want: ScopeKindOrganization, ok: true},
		{input: " PROJECT ", want: ScopeKindProject, ok: true},
		{input: "ticket", ok: false},
	}

	for _, tc := range tests {
		got, err := ParseScopeKind(tc.input)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("ParseScopeKind(%q) = (%q, %v), want (%q, nil)", tc.input, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("ParseScopeKind(%q) expected error", tc.input)
		}
	}
}

func TestParseKind(t *testing.T) {
	tests := []struct {
		input string
		want  Kind
		ok    bool
	}{
		{input: "", want: KindOpaque, ok: true},
		{input: " OPAQUE ", want: KindOpaque, ok: true},
		{input: "json", ok: false},
	}

	for _, tc := range tests {
		got, err := ParseKind(tc.input)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("ParseKind(%q) = (%q, %v), want (%q, nil)", tc.input, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("ParseKind(%q) expected error", tc.input)
		}
	}
}

func TestParseBindingScopeKind(t *testing.T) {
	tests := []struct {
		input string
		want  BindingScopeKind
		ok    bool
	}{
		{input: "organization", want: BindingScopeKindOrganization, ok: true},
		{input: "project", want: BindingScopeKindProject, ok: true},
		{input: "workflow", want: BindingScopeKindWorkflow, ok: true},
		{input: "agent", want: BindingScopeKindAgent, ok: true},
		{input: "ticket", want: BindingScopeKindTicket, ok: true},
		{input: "machine", ok: false},
	}

	for _, tc := range tests {
		got, err := ParseBindingScopeKind(tc.input)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("ParseBindingScopeKind(%q) = (%q, %v), want (%q, nil)", tc.input, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("ParseBindingScopeKind(%q) expected error", tc.input)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	got, err := NormalizeName(" gh_token ")
	if err != nil {
		t.Fatalf("NormalizeName() error = %v", err)
	}
	if got != "GH_TOKEN" {
		t.Fatalf("NormalizeName() = %q", got)
	}
	if _, err := NormalizeName("bad-name"); err == nil {
		t.Fatal("NormalizeName() expected error for invalid name")
	}
}

func TestParseBindingKeys(t *testing.T) {
	got, err := ParseBindingKeys([]string{" gh_token ", "OPENAI_API_KEY", "GH_TOKEN"})
	if err != nil {
		t.Fatalf("ParseBindingKeys() error = %v", err)
	}
	if len(got) != 2 || got[0] != "GH_TOKEN" || got[1] != "OPENAI_API_KEY" {
		t.Fatalf("ParseBindingKeys() = %v", got)
	}
	if _, err := ParseBindingKeys(nil); err == nil {
		t.Fatal("ParseBindingKeys(nil) expected error")
	}
	if _, err := ParseBindingKeys([]string{"bad-key"}); err == nil {
		t.Fatal("ParseBindingKeys(invalid) expected error")
	}
}

func TestBindingKeysFromCandidates(t *testing.T) {
	got, err := BindingKeysFromCandidates([]Candidate{
		{Binding: Binding{BindingKey: " gh_token "}},
		{Binding: Binding{BindingKey: "OPENAI_API_KEY"}},
		{Binding: Binding{BindingKey: "GH_TOKEN"}},
	})
	if err != nil {
		t.Fatalf("BindingKeysFromCandidates() error = %v", err)
	}
	if len(got) != 2 || got[0] != "GH_TOKEN" || got[1] != "OPENAI_API_KEY" {
		t.Fatalf("BindingKeysFromCandidates() = %v", got)
	}

	empty, err := BindingKeysFromCandidates(nil)
	if err != nil {
		t.Fatalf("BindingKeysFromCandidates(nil) error = %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("BindingKeysFromCandidates(nil) = %v, want empty", empty)
	}
}

func TestDefaultCipherSeedAndRedactValue(t *testing.T) {
	seed := DefaultCipherSeed("postgres://example")
	decoded, err := base64.StdEncoding.DecodeString(seed)
	if err != nil {
		t.Fatalf("DefaultCipherSeed() returned invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("DefaultCipherSeed() decoded length = %d, want 32", len(decoded))
	}

	if got := RedactValue(""); got != "" {
		t.Fatalf("RedactValue(\"\") = %q", got)
	}
	if got := RedactValue("abcd"); got != "****" {
		t.Fatalf("RedactValue(\"abcd\") = %q", got)
	}
	if got := RedactValue("0123456789"); got != "012345...6789" {
		t.Fatalf("RedactValue() = %q", got)
	}
}

func TestProjectIDScopeHelpers(t *testing.T) {
	projectID := uuid.New()
	if got := ProjectIDForSecretScope(ScopeKindOrganization, projectID); got != uuid.Nil {
		t.Fatalf("ProjectIDForSecretScope(organization) = %s, want nil uuid", got)
	}
	if got := ProjectIDForSecretScope(ScopeKindProject, projectID); got != projectID {
		t.Fatalf("ProjectIDForSecretScope(project) = %s, want %s", got, projectID)
	}
	if got := ProjectIDForBindingScope(BindingScopeKindOrganization, projectID); got != uuid.Nil {
		t.Fatalf("ProjectIDForBindingScope(organization) = %s, want nil uuid", got)
	}
	if got := ProjectIDForBindingScope(BindingScopeKindTicket, projectID); got != projectID {
		t.Fatalf("ProjectIDForBindingScope(ticket) = %s, want %s", got, projectID)
	}
}

func TestResolutionRank(t *testing.T) {
	if got := ResolutionRank(BindingScopeKindTicket); got != 0 {
		t.Fatalf("ResolutionRank(ticket) = %d, want 0", got)
	}
	if got := ResolutionRank(BindingScopeKindWorkflow); got != 1 {
		t.Fatalf("ResolutionRank(workflow) = %d, want 1", got)
	}
	if got := ResolutionRank(BindingScopeKindAgent); got != 1 {
		t.Fatalf("ResolutionRank(agent) = %d, want 1", got)
	}
	if got := ResolutionRank(BindingScopeKindProject); got != 2 {
		t.Fatalf("ResolutionRank(project) = %d, want 2", got)
	}
	if got := ResolutionRank(BindingScopeKindOrganization); got != 3 {
		t.Fatalf("ResolutionRank(organization) = %d, want 3", got)
	}
	if got := ResolutionRank(BindingScopeKind("unknown")); got != 99 {
		t.Fatalf("ResolutionRank(unknown) = %d, want 99", got)
	}
}

func TestSelectBindingsHonorsPrecedence(t *testing.T) {
	secretID := uuid.New()
	keys := []string{"GH_TOKEN"}
	selected, missing, err := SelectBindings(keys, []Candidate{
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindOrganization}, Secret: Secret{ID: secretID, Name: "ORG_TOKEN", Scope: ScopeKindOrganization}},
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindProject}, Secret: Secret{ID: secretID, Name: "PROJECT_TOKEN", Scope: ScopeKindProject}},
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindTicket}, Secret: Secret{ID: secretID, Name: "TICKET_TOKEN", Scope: ScopeKindProject}},
	})
	if err != nil {
		t.Fatalf("SelectBindings() error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if len(selected) != 1 || selected[0].Binding.Scope != BindingScopeKindTicket {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestSelectBindingsFallsBackWhenHigherScopeSecretDisabled(t *testing.T) {
	disabledAt := time.Now().UTC()
	selected, missing, err := SelectBindings([]string{"OPENAI_API_KEY"}, []Candidate{
		{Binding: Binding{BindingKey: "OPENAI_API_KEY", Scope: BindingScopeKindTicket}, Secret: Secret{ID: uuid.New(), Name: "TICKET", Scope: ScopeKindProject, DisabledAt: &disabledAt}},
		{Binding: Binding{BindingKey: "OPENAI_API_KEY", Scope: BindingScopeKindProject}, Secret: Secret{ID: uuid.New(), Name: "PROJECT", Scope: ScopeKindProject}},
	})
	if err != nil {
		t.Fatalf("SelectBindings() error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if len(selected) != 1 || selected[0].Binding.Scope != BindingScopeKindProject {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestSelectBindingsReturnsMissingAndAllowsSameSecretAtEqualPrecedence(t *testing.T) {
	sharedSecretID := uuid.New()
	selected, missing, err := SelectBindings([]string{"GH_TOKEN", "OPENAI_API_KEY"}, []Candidate{
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindWorkflow}, Secret: Secret{ID: sharedSecretID, Name: "WORKFLOW", Scope: ScopeKindProject}},
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindAgent}, Secret: Secret{ID: sharedSecretID, Name: "AGENT", Scope: ScopeKindProject}},
	})
	if err != nil {
		t.Fatalf("SelectBindings() error = %v", err)
	}
	if len(selected) != 1 || selected[0].BindingKey != "GH_TOKEN" {
		t.Fatalf("selected = %#v", selected)
	}
	if len(missing) != 1 || missing[0] != "OPENAI_API_KEY" {
		t.Fatalf("missing = %v", missing)
	}
}

func TestSelectBindingsAllowsSameSecretWithinSameScope(t *testing.T) {
	sharedSecretID := uuid.New()
	selected, missing, err := SelectBindings([]string{"GH_TOKEN"}, []Candidate{
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindProject}, Secret: Secret{ID: sharedSecretID, Name: "ZZZ_PROJECT", Scope: ScopeKindProject}},
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindProject}, Secret: Secret{ID: sharedSecretID, Name: "AAA_PROJECT", Scope: ScopeKindProject}},
	})
	if err != nil {
		t.Fatalf("SelectBindings() error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if len(selected) != 1 || selected[0].Secret.Name != "AAA_PROJECT" {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestSelectBindingsMarksAllDisabledCandidatesMissing(t *testing.T) {
	disabledAt := time.Now().UTC()
	selected, missing, err := SelectBindings([]string{"GH_TOKEN"}, []Candidate{
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindProject}, Secret: Secret{ID: uuid.New(), Name: "PROJECT", Scope: ScopeKindProject, DisabledAt: &disabledAt}},
	})
	if err != nil {
		t.Fatalf("SelectBindings() error = %v", err)
	}
	if len(selected) != 0 {
		t.Fatalf("selected = %#v, want empty", selected)
	}
	if len(missing) != 1 || missing[0] != "GH_TOKEN" {
		t.Fatalf("missing = %v", missing)
	}
}

func TestSelectBindingsRejectsWorkflowAgentConflict(t *testing.T) {
	_, _, err := SelectBindings([]string{"GH_TOKEN"}, []Candidate{
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindWorkflow}, Secret: Secret{ID: uuid.New(), Name: "WORKFLOW", Scope: ScopeKindProject}},
		{Binding: Binding{BindingKey: "GH_TOKEN", Scope: BindingScopeKindAgent}, Secret: Secret{ID: uuid.New(), Name: "AGENT", Scope: ScopeKindProject}},
	})
	if !errors.Is(err, ErrResolutionScopeConflict) {
		t.Fatalf("SelectBindings() err = %v, want ErrResolutionScopeConflict", err)
	}
}
