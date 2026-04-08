package catalog

import "testing"

func TestMaskMachineEnvVars(t *testing.T) {
	raw := []string{
		"CUDA_VISIBLE_DEVICES=0",
		"OPENAI_API_KEY=sk-live-1234",
		"WEBHOOK_SECRET=hook-secret",
		"EMPTY_SECRET=",
	}

	got := MaskMachineEnvVars(raw)

	want := []string{
		"CUDA_VISIBLE_DEVICES=0",
		"OPENAI_API_KEY=[redacted]",
		"WEBHOOK_SECRET=[redacted]",
		"EMPTY_SECRET=",
	}
	if len(got) != len(want) {
		t.Fatalf("MaskMachineEnvVars() len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for index, wantItem := range want {
		if got[index] != wantItem {
			t.Fatalf("MaskMachineEnvVars()[%d] = %q, want %q", index, got[index], wantItem)
		}
	}
}

func TestMergeMaskedMachineEnvVarsPreservesExistingSensitiveValues(t *testing.T) {
	current := []string{
		"OPENAI_API_KEY=sk-live-1234",
		"CUDA_VISIBLE_DEVICES=0",
	}
	requested := []string{
		"OPENAI_API_KEY=[redacted]",
		"CUDA_VISIBLE_DEVICES=1",
	}

	got := MergeMaskedMachineEnvVars(current, requested)

	want := []string{
		"OPENAI_API_KEY=sk-live-1234",
		"CUDA_VISIBLE_DEVICES=1",
	}
	for index, wantItem := range want {
		if got[index] != wantItem {
			t.Fatalf("MergeMaskedMachineEnvVars()[%d] = %q, want %q", index, got[index], wantItem)
		}
	}
}

func TestCountLegacyProviderInlineSecretBindings(t *testing.T) {
	providers := []AgentProvider{
		{
			AdapterType: AgentProviderAdapterTypeCodexAppServer,
			AuthConfig: map[string]any{
				"base_url":       "http://localhost:4318",
				"openai_api_key": "legacy-secret",
			},
		},
		{
			AdapterType: AgentProviderAdapterTypeGeminiCLI,
			AuthConfig: map[string]any{
				"secret_refs": map[string]any{
					"GEMINI_API_KEY": "PROJECT_GEMINI_KEY",
				},
			},
		},
	}

	providersWithLegacy, legacyBindingCount := CountLegacyProviderInlineSecretBindings(providers)
	if providersWithLegacy != 1 || legacyBindingCount != 1 {
		t.Fatalf(
			"CountLegacyProviderInlineSecretBindings() = (%d, %d), want (1, 1)",
			providersWithLegacy,
			legacyBindingCount,
		)
	}
}

func TestCountSensitiveMachineEnvVars(t *testing.T) {
	machines := []Machine{
		{EnvVars: []string{"OPENAI_API_KEY=sk-live-1234", "CUDA_VISIBLE_DEVICES=0"}},
		{EnvVars: []string{"WEBHOOK_SECRET=hook-secret"}},
		{EnvVars: []string{"PATH=/usr/bin"}},
	}

	machinesWithSensitive, sensitiveEnvVarCount := CountSensitiveMachineEnvVars(machines)
	if machinesWithSensitive != 2 || sensitiveEnvVarCount != 2 {
		t.Fatalf(
			"CountSensitiveMachineEnvVars() = (%d, %d), want (2, 2)",
			machinesWithSensitive,
			sensitiveEnvVarCount,
		)
	}
}
