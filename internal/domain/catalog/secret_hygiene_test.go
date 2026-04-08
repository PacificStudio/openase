package catalog

import "testing"

func TestMaskMachineEnvVars(t *testing.T) {
	raw := []string{
		"CUDA_VISIBLE_DEVICES=0",
		"OPENAI_API_KEY=sk-live-1234",
		"WEBHOOK_SECRET=hook-secret",
		"EMPTY_SECRET=",
		"  INVALID ENTRY  ",
		" AUTH_TOKEN = bearer-token ",
	}

	got := MaskMachineEnvVars(raw)

	want := []string{
		"CUDA_VISIBLE_DEVICES=0",
		"OPENAI_API_KEY=[redacted]",
		"WEBHOOK_SECRET=[redacted]",
		"EMPTY_SECRET=",
		"INVALID ENTRY",
		"AUTH_TOKEN=[redacted]",
	}
	if len(got) != len(want) {
		t.Fatalf("MaskMachineEnvVars() len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for index, wantItem := range want {
		if got[index] != wantItem {
			t.Fatalf("MaskMachineEnvVars()[%d] = %q, want %q", index, got[index], wantItem)
		}
	}

	if got := MaskMachineEnvVars(nil); got != nil {
		t.Fatalf("MaskMachineEnvVars(nil) = %+v, want nil", got)
	}
}

func TestMergeMaskedMachineEnvVarsPreservesExistingSensitiveValues(t *testing.T) {
	current := []string{
		"OPENAI_API_KEY=sk-live-1234",
		"CUDA_VISIBLE_DEVICES=0",
		"BROKEN_ENTRY",
	}
	requested := []string{
		"OPENAI_API_KEY=[redacted]",
		"CUDA_VISIBLE_DEVICES=1",
		"WEBHOOK_SECRET=[redacted]",
		"INVALID ENTRY",
	}

	got := MergeMaskedMachineEnvVars(current, requested)

	want := []string{
		"OPENAI_API_KEY=sk-live-1234",
		"CUDA_VISIBLE_DEVICES=1",
		"WEBHOOK_SECRET=[redacted]",
		"INVALID ENTRY",
	}
	for index, wantItem := range want {
		if got[index] != wantItem {
			t.Fatalf("MergeMaskedMachineEnvVars()[%d] = %q, want %q", index, got[index], wantItem)
		}
	}

	if got := MergeMaskedMachineEnvVars(current, nil); got != nil {
		t.Fatalf("MergeMaskedMachineEnvVars(nil) = %+v, want nil", got)
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

func TestSensitiveMachineEnvVarHelpers(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		got := SensitiveMachineEnvVarCount([]string{
			"OPENAI_API_KEY=sk-live-1234",
			"DATABASE_DSN=postgres://secret",
			"PATH=/usr/bin",
			"INVALID ENTRY",
		})
		if got != 2 {
			t.Fatalf("SensitiveMachineEnvVarCount() = %d, want 2", got)
		}
	})

	t.Run("sensitive keys", func(t *testing.T) {
		cases := []struct {
			key  string
			want bool
		}{
			{key: "OPENAI_API_KEY", want: true},
			{key: "database_dsn", want: true},
			{key: "session_key", want: true},
			{key: "PATH", want: false},
			{key: "   ", want: false},
		}
		for _, tc := range cases {
			if got := IsSensitiveMachineEnvVarKey(tc.key); got != tc.want {
				t.Fatalf("IsSensitiveMachineEnvVarKey(%q) = %t, want %t", tc.key, got, tc.want)
			}
		}
	})

	t.Run("split env vars", func(t *testing.T) {
		cases := []struct {
			raw       string
			wantKey   string
			wantValue string
			wantOK    bool
		}{
			{raw: "OPENAI_API_KEY=sk-live-1234", wantKey: "OPENAI_API_KEY", wantValue: "sk-live-1234", wantOK: true},
			{raw: " AUTH_TOKEN = bearer ", wantKey: "AUTH_TOKEN", wantValue: " bearer", wantOK: true},
			{raw: "NO_EQUALS", wantOK: false},
			{raw: " =missing-key", wantOK: false},
			{raw: "   ", wantOK: false},
		}
		for _, tc := range cases {
			gotKey, gotValue, gotOK := splitMachineEnvVar(tc.raw)
			if gotKey != tc.wantKey || gotValue != tc.wantValue || gotOK != tc.wantOK {
				t.Fatalf(
					"splitMachineEnvVar(%q) = (%q, %q, %t), want (%q, %q, %t)",
					tc.raw,
					gotKey,
					gotValue,
					gotOK,
					tc.wantKey,
					tc.wantValue,
					tc.wantOK,
				)
			}
		}
	})
}
