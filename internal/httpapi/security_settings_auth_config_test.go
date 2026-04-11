package httpapi

import (
	"strings"
	"testing"
	"time"

	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
)

func TestDraftOIDCConfigFromRequestParsesSessionPolicy(t *testing.T) {
	t.Parallel()

	current := iam.AccessControlState{
		Status: iam.AccessControlStatusDraft,
		Draft: &iam.DraftOIDCConfig{
			Claims:        iam.DefaultDraftOIDCConfig().Claims,
			Scopes:        []string{"openid", "profile", "email"},
			SessionPolicy: iam.OIDCSessionPolicy{SessionTTL: 8 * time.Hour, SessionIdleTTL: 30 * time.Minute},
		},
	}

	draft, err := draftOIDCConfigFromRequest(rawSecurityOIDCDraftRequest{
		SessionTTL:     "2h",
		SessionIdleTTL: "15m",
	}, current)
	if err != nil {
		t.Fatalf("draftOIDCConfigFromRequest() error = %v", err)
	}

	if draft.SessionPolicy.SessionTTL != 2*time.Hour {
		t.Fatalf("session ttl = %s, want 2h", draft.SessionPolicy.SessionTTL)
	}
	if draft.SessionPolicy.SessionIdleTTL != 15*time.Minute {
		t.Fatalf("session idle ttl = %s, want 15m", draft.SessionPolicy.SessionIdleTTL)
	}
}

func TestDraftOIDCConfigFromRequestPreservesExistingSessionPolicyWhenOmitted(t *testing.T) {
	t.Parallel()

	current := iam.AccessControlState{
		Status: iam.AccessControlStatusDraft,
		Draft: &iam.DraftOIDCConfig{
			Claims: iam.DefaultDraftOIDCConfig().Claims,
			Scopes: []string{"openid", "profile", "email"},
			SessionPolicy: iam.OIDCSessionPolicy{
				SessionTTL:     12 * time.Hour,
				SessionIdleTTL: 45 * time.Minute,
			},
		},
	}

	draft, err := draftOIDCConfigFromRequest(rawSecurityOIDCDraftRequest{}, current)
	if err != nil {
		t.Fatalf("draftOIDCConfigFromRequest() error = %v", err)
	}

	if draft.SessionPolicy != current.Draft.SessionPolicy {
		t.Fatalf("session policy = %+v, want %+v", draft.SessionPolicy, current.Draft.SessionPolicy)
	}
}

func TestDraftOIDCConfigFromRequestRejectsIdleTTLAboveSessionTTL(t *testing.T) {
	t.Parallel()

	current := iam.AccessControlState{
		Status: iam.AccessControlStatusDraft,
		Draft: &iam.DraftOIDCConfig{
			Claims:        iam.DefaultDraftOIDCConfig().Claims,
			Scopes:        []string{"openid", "profile", "email"},
			SessionPolicy: iam.DefaultDraftOIDCConfig().SessionPolicy,
		},
	}

	_, err := draftOIDCConfigFromRequest(rawSecurityOIDCDraftRequest{
		SessionTTL:     "1h",
		SessionIdleTTL: "2h",
	}, current)
	if err == nil || !strings.Contains(err.Error(), "session_idle_ttl must not exceed session_ttl") {
		t.Fatalf("expected readable ttl validation error, got %v", err)
	}
}
