package catalog

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildAgentProviderAuthConfigStoresPlainConfigAndSecretRefs(t *testing.T) {
	authConfig, err := BuildAgentProviderAuthConfig(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{
			"base_url":       "http://localhost:4318",
			"openai_api_key": "legacy-secret",
		},
		[]AgentProviderSecretBindingInput{
			{EnvVarKey: "openai_api_key", BindingKey: "provider_openai_key"},
		},
	)
	if err != nil {
		t.Fatalf("BuildAgentProviderAuthConfig() error = %v", err)
	}

	if got := authConfig["base_url"]; got != "http://localhost:4318" {
		t.Fatalf("base_url = %#v, want %q", got, "http://localhost:4318")
	}
	if got := authConfig["openai_api_key"]; got != "legacy-secret" {
		t.Fatalf("legacy openai_api_key = %#v, want preserved legacy inline value", got)
	}

	secretRefs, ok := authConfig["secret_refs"].(map[string]any)
	if !ok {
		t.Fatalf("secret_refs type = %T, want map[string]any", authConfig["secret_refs"])
	}
	if got := secretRefs["OPENAI_API_KEY"]; got != "PROVIDER_OPENAI_KEY" {
		t.Fatalf("secret_refs[OPENAI_API_KEY] = %#v, want %q", got, "PROVIDER_OPENAI_KEY")
	}
}

func TestVisibleAgentProviderAuthConfigOmitsSecretValues(t *testing.T) {
	visible := VisibleAgentProviderAuthConfig(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{
			"base_url":       "http://localhost:4318",
			"token":          "legacy-token",
			"openai_api_key": "legacy-openai-key",
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROVIDER_OPENAI_KEY",
			},
		},
	)

	if got := visible["base_url"]; got != "http://localhost:4318" {
		t.Fatalf("visible base_url = %#v, want %q", got, "http://localhost:4318")
	}
	if _, ok := visible["token"]; ok {
		t.Fatalf("visible auth config unexpectedly included token: %#v", visible)
	}
	if _, ok := visible["openai_api_key"]; ok {
		t.Fatalf("visible auth config unexpectedly included openai_api_key: %#v", visible)
	}
	if _, ok := visible["secret_refs"]; ok {
		t.Fatalf("visible auth config unexpectedly included secret_refs: %#v", visible)
	}
}

func TestAgentProviderSecretBindingsDescribeExplicitLegacyAndDefaultEntries(t *testing.T) {
	bindings := AgentProviderSecretBindings(
		AgentProviderAdapterTypeGeminiCLI,
		map[string]any{
			"region": "us-west-2",
			"token":  "legacy-token",
			"secret_refs": map[string]any{
				"GEMINI_API_KEY": "PROJECT_GEMINI_KEY",
			},
		},
	)

	got := map[string]AgentProviderSecretBinding{}
	for _, item := range bindings {
		got[item.EnvVarKey] = item
	}

	if binding := got["GEMINI_API_KEY"]; !binding.Configured ||
		binding.BindingKey != "PROJECT_GEMINI_KEY" ||
		binding.Source != AgentProviderSecretBindingSourceBinding {
		t.Fatalf("GEMINI_API_KEY binding = %#v", binding)
	}
	if binding := got["GOOGLE_API_KEY"]; binding.Configured ||
		binding.BindingKey != "GOOGLE_API_KEY" ||
		binding.Source != AgentProviderSecretBindingSourceDefault {
		t.Fatalf("GOOGLE_API_KEY binding = %#v", binding)
	}
	if binding := got["TOKEN"]; !binding.Configured ||
		binding.BindingKey != "TOKEN" ||
		binding.Source != AgentProviderSecretBindingSourceLegacyAuthConfig {
		t.Fatalf("TOKEN binding = %#v", binding)
	}
}

func TestMergeAgentProviderAuthConfigPreservesStoredRefsWhenPatchOmitsSecretBindings(t *testing.T) {
	merged, err := MergeAgentProviderAuthConfig(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{
			"base_url": "http://localhost:4318",
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
		&map[string]any{
			"base_url": "http://localhost:8080",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("MergeAgentProviderAuthConfig() error = %v", err)
	}

	if got := merged["base_url"]; got != "http://localhost:8080" {
		t.Fatalf("base_url = %#v, want %q", got, "http://localhost:8080")
	}
	secretRefs, ok := merged["secret_refs"].(map[string]any)
	if !ok {
		t.Fatalf("secret_refs type = %T, want map[string]any", merged["secret_refs"])
	}
	if got := secretRefs["OPENAI_API_KEY"]; got != "PROJECT_OPENAI_KEY" {
		t.Fatalf("secret_refs[OPENAI_API_KEY] = %#v, want %q", got, "PROJECT_OPENAI_KEY")
	}
}

func TestMergeAgentProviderAuthConfigPrefersExplicitBindingsOverInlineSecrets(t *testing.T) {
	merged, err := MergeAgentProviderAuthConfig(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{
			"base_url":       "http://localhost:4318",
			"openai_api_key": "legacy-secret",
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "OLD_OPENAI_KEY",
			},
		},
		&map[string]any{
			"base_url":       "http://localhost:8080",
			"openai_api_key": "updated-inline-secret",
		},
		&[]AgentProviderSecretBindingInput{
			{EnvVarKey: "openai_api_key", BindingKey: "project_openai_key"},
		},
	)
	if err != nil {
		t.Fatalf("MergeAgentProviderAuthConfig() error = %v", err)
	}

	if got := merged["base_url"]; got != "http://localhost:8080" {
		t.Fatalf("base_url = %#v, want %q", got, "http://localhost:8080")
	}
	if _, ok := merged["openai_api_key"]; ok {
		t.Fatalf("merge kept inline openai_api_key even though explicit binding now exists: %#v", merged)
	}

	secretRefs, ok := merged["secret_refs"].(map[string]any)
	if !ok {
		t.Fatalf("secret_refs type = %T, want map[string]any", merged["secret_refs"])
	}
	if got := secretRefs["OPENAI_API_KEY"]; got != "PROJECT_OPENAI_KEY" {
		t.Fatalf("secret_refs[OPENAI_API_KEY] = %#v, want %q", got, "PROJECT_OPENAI_KEY")
	}
}

func TestBuildAgentProviderAuthConfigRejectsInvalidSecretBindings(t *testing.T) {
	_, err := BuildAgentProviderAuthConfig(
		AgentProviderAdapterTypeCustom,
		nil,
		[]AgentProviderSecretBindingInput{
			{EnvVarKey: "123", BindingKey: "valid_name"},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "secret_bindings[0].env_var_key") {
		t.Fatalf("BuildAgentProviderAuthConfig() error = %v, want env_var_key validation error", err)
	}
}

func TestMergeAgentProviderAuthConfigRejectsInvalidSecretBindings(t *testing.T) {
	_, err := MergeAgentProviderAuthConfig(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{"base_url": "http://localhost:4318"},
		nil,
		&[]AgentProviderSecretBindingInput{
			{EnvVarKey: "openai_api_key", BindingKey: "123"},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "secret_bindings[0].binding_key") {
		t.Fatalf("MergeAgentProviderAuthConfig() error = %v, want binding_key validation error", err)
	}
}

func TestParseCreateAgentProviderRejectsInvalidSecretBindings(t *testing.T) {
	_, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		MachineID:   uuid.New().String(),
		Name:        "Codex",
		AdapterType: "codex-app-server",
		CliCommand:  "codex",
		ModelName:   "gpt-5.4",
		SecretBindings: []AgentProviderSecretBindingInput{
			{EnvVarKey: "openai_api_key", BindingKey: "123"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "secret_bindings[0].binding_key") {
		t.Fatalf("ParseCreateAgentProvider() error = %v, want binding_key validation error", err)
	}
}

func TestComposeAgentProviderAuthConfigSkipsReservedSecretRefsField(t *testing.T) {
	got := composeAgentProviderAuthConfig(
		map[string]any{
			"base_url":    "http://localhost:4318",
			"secret_refs": "discard-plain",
		},
		map[string]any{
			"token":       "legacy-token",
			"secret_refs": "discard-legacy",
		},
		map[string]string{},
	)

	if got["base_url"] != "http://localhost:4318" || got["token"] != "legacy-token" {
		t.Fatalf("composeAgentProviderAuthConfig() = %#v", got)
	}
	if _, ok := got["secret_refs"]; ok {
		t.Fatalf("composeAgentProviderAuthConfig() unexpectedly kept reserved field: %#v", got)
	}
}

func TestParseAgentProviderSecretBindingInputsAllowsEmptySlice(t *testing.T) {
	got, err := parseAgentProviderSecretBindingInputs(nil)
	if err != nil {
		t.Fatalf("parseAgentProviderSecretBindingInputs() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("parseAgentProviderSecretBindingInputs() = %#v, want empty map", got)
	}
}

func TestAgentProviderExplicitSecretRefsDropsInvalidStoredEntries(t *testing.T) {
	got := AgentProviderExplicitSecretRefs(map[string]any{
		"secret_refs": map[string]string{
			"openai_api_key": "project_openai_key",
			"123":            "SHOULD_BE_DROPPED",
			"BAD_TARGET":     "123",
		},
	})

	want := map[string]string{
		"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
	}
	if len(got) != len(want) {
		t.Fatalf("AgentProviderExplicitSecretRefs() len = %d, want %d (%#v)", len(got), len(want), got)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("AgentProviderExplicitSecretRefs()[%q] = %q, want %q", key, got[key], wantValue)
		}
	}
}

func TestParseStoredAgentProviderSecretRefsHandlesSupportedShapes(t *testing.T) {
	testCases := []struct {
		name string
		raw  map[string]any
		want map[string]string
	}{
		{
			name: "empty raw map",
			raw:  nil,
			want: map[string]string{},
		},
		{
			name: "missing field",
			raw:  map[string]any{"base_url": "http://localhost"},
			want: map[string]string{},
		},
		{
			name: "nil field",
			raw: map[string]any{
				"secret_refs": nil,
			},
			want: map[string]string{},
		},
		{
			name: "map any with filtering",
			raw: map[string]any{
				"secret_refs": map[string]any{
					"openai api key": "project-openai-key",
					"":               "IGNORED",
					"GOOD_KEY":       "123",
				},
			},
			want: map[string]string{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
		{
			name: "unsupported stored type",
			raw: map[string]any{
				"secret_refs": []string{"OPENAI_API_KEY"},
			},
			want: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStoredAgentProviderSecretRefs(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("parseStoredAgentProviderSecretRefs() len = %d, want %d (%#v)", len(got), len(tc.want), got)
			}
			for key, wantValue := range tc.want {
				if got[key] != wantValue {
					t.Fatalf("parseStoredAgentProviderSecretRefs()[%q] = %q, want %q", key, got[key], wantValue)
				}
			}
		})
	}
}

func TestAgentProviderSecretBindingsForCustomAdapter(t *testing.T) {
	bindings := AgentProviderSecretBindings(
		AgentProviderAdapterTypeCustom,
		map[string]any{
			"api key": "legacy-key",
			"secret_refs": map[string]any{
				"secondary-token": "secondary_binding",
			},
			"region": "us-west-2",
		},
	)

	if len(bindings) != 2 {
		t.Fatalf("AgentProviderSecretBindings() len = %d, want 2 (%#v)", len(bindings), bindings)
	}
	if bindings[0].EnvVarKey != "API_KEY" || bindings[0].Source != AgentProviderSecretBindingSourceLegacyAuthConfig || !bindings[0].Configured {
		t.Fatalf("first binding = %#v", bindings[0])
	}
	if bindings[1].EnvVarKey != "SECONDARY_TOKEN" || bindings[1].BindingKey != "SECONDARY_BINDING" || bindings[1].Source != AgentProviderSecretBindingSourceBinding || !bindings[1].Configured {
		t.Fatalf("second binding = %#v", bindings[1])
	}
}

func TestAgentProviderSecretBindingsPreferExplicitBindingWhenLegacySecretUsesSameEnvVar(t *testing.T) {
	bindings := AgentProviderSecretBindings(
		AgentProviderAdapterTypeCodexAppServer,
		map[string]any{
			"openai_api_key": "legacy-inline-key",
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
	)

	if len(bindings) != 1 {
		t.Fatalf("AgentProviderSecretBindings() len = %d, want 1 (%#v)", len(bindings), bindings)
	}
	if got := bindings[0]; got.EnvVarKey != "OPENAI_API_KEY" || got.BindingKey != "PROJECT_OPENAI_KEY" || got.Source != AgentProviderSecretBindingSourceBinding || !got.Configured {
		t.Fatalf("binding = %#v", got)
	}
}

func TestParseProviderBindingNameAndNormalizeProviderEnvKey(t *testing.T) {
	if got := normalizeProviderEnvKey("  openai-api key  "); got != "OPENAI_API_KEY" {
		t.Fatalf("normalizeProviderEnvKey() = %q, want OPENAI_API_KEY", got)
	}
	if got := normalizeProviderEnvKey("___"); got != "" {
		t.Fatalf("normalizeProviderEnvKey() = %q, want empty string", got)
	}

	if got, err := parseProviderBindingName(" provider-openai-key "); err != nil || got != "PROVIDER_OPENAI_KEY" {
		t.Fatalf("parseProviderBindingName() = (%q, %v), want (PROVIDER_OPENAI_KEY, nil)", got, err)
	}
	if _, err := parseProviderBindingName("123"); err == nil {
		t.Fatal("parseProviderBindingName() expected validation error for numeric binding name")
	}
}

func TestShouldTreatProviderAuthConfigKeyAsSecretAndRequiredEnvVars(t *testing.T) {
	if shouldTreatProviderAuthConfigKeyAsSecret(AgentProviderAdapterTypeCustom, "   ") {
		t.Fatal("did not expect empty normalized key to be treated as secret")
	}
	if !shouldTreatProviderAuthConfigKeyAsSecret(AgentProviderAdapterTypeGeminiCLI, "google_api_key") {
		t.Fatal("expected gemini google_api_key to be treated as secret")
	}
	if !shouldTreatProviderAuthConfigKeyAsSecret(AgentProviderAdapterTypeCustom, "client_secret") {
		t.Fatal("expected client_secret to be treated as secret")
	}
	if shouldTreatProviderAuthConfigKeyAsSecret(AgentProviderAdapterTypeCustom, "region") {
		t.Fatal("did not expect region to be treated as secret")
	}

	if got := requiredProviderSecretEnvVars(AgentProviderAdapterTypeClaudeCodeCLI); len(got) != 1 || got[0] != "ANTHROPIC_API_KEY" {
		t.Fatalf("requiredProviderSecretEnvVars(claude) = %#v", got)
	}
	if got := requiredProviderSecretEnvVars(AgentProviderAdapterTypeCodexAppServer); len(got) != 1 || got[0] != "OPENAI_API_KEY" {
		t.Fatalf("requiredProviderSecretEnvVars(codex) = %#v", got)
	}
	if got := requiredProviderSecretEnvVars(AgentProviderAdapterTypeGeminiCLI); len(got) != 2 || got[0] != "GEMINI_API_KEY" || got[1] != "GOOGLE_API_KEY" {
		t.Fatalf("requiredProviderSecretEnvVars(gemini) = %#v", got)
	}
	if got := requiredProviderSecretEnvVars(AgentProviderAdapterTypeCustom); got != nil {
		t.Fatalf("requiredProviderSecretEnvVars(custom) = %#v, want nil", got)
	}
}
