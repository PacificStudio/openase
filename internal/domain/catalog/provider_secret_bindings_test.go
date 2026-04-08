package catalog

import "testing"

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
