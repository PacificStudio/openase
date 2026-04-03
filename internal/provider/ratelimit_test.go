package provider

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseClaudeCodeRateLimit(t *testing.T) {
	rateLimit, err := ParseClaudeCodeRateLimit([]byte(`{
		"status":"allowed",
		"resetsAt":1775037600,
		"rateLimitType":"five_hour",
		"utilization":0.93,
		"surpassedThreshold":0.75,
		"overageStatus":"rejected",
		"overageDisabledReason":"org_level_disabled",
		"isUsingOverage":false
	}`))
	if err != nil {
		t.Fatalf("ParseClaudeCodeRateLimit returned error: %v", err)
	}
	if rateLimit == nil {
		t.Fatal("ParseClaudeCodeRateLimit returned nil")
	}
	if rateLimit.Provider != CLIRateLimitProviderClaudeCode {
		t.Fatalf("provider = %q", rateLimit.Provider)
	}
	if rateLimit.ClaudeCode == nil {
		t.Fatal("ClaudeCode payload = nil")
	}
	if rateLimit.ClaudeCode.Status != "allowed" || rateLimit.ClaudeCode.RateLimitType != "five_hour" {
		t.Fatalf("ClaudeCode payload = %+v", rateLimit.ClaudeCode)
	}
	if rateLimit.ClaudeCode.ResetsAt == nil || !rateLimit.ClaudeCode.ResetsAt.Equal(time.Unix(1775037600, 0).UTC()) {
		t.Fatalf("resets_at = %+v", rateLimit.ClaudeCode.ResetsAt)
	}
	if rateLimit.ClaudeCode.Utilization == nil || *rateLimit.ClaudeCode.Utilization != 0.93 {
		t.Fatalf("utilization = %+v", rateLimit.ClaudeCode.Utilization)
	}
	if rateLimit.ClaudeCode.SurpassedThreshold == nil || *rateLimit.ClaudeCode.SurpassedThreshold != 0.75 {
		t.Fatalf("surpassed_threshold = %+v", rateLimit.ClaudeCode.SurpassedThreshold)
	}
	if rateLimit.ClaudeCode.IsUsingOverage == nil || *rateLimit.ClaudeCode.IsUsingOverage {
		t.Fatalf("is_using_overage = %+v", rateLimit.ClaudeCode.IsUsingOverage)
	}
	if rateLimit.Raw["status"] != "allowed" {
		t.Fatalf("raw payload = %+v", rateLimit.Raw)
	}
}

func TestParseClaudeCodeRateLimitEmptyPayload(t *testing.T) {
	rateLimit, err := ParseClaudeCodeRateLimit(nil)
	if err != nil {
		t.Fatalf("ParseClaudeCodeRateLimit(nil) returned error: %v", err)
	}
	if rateLimit != nil {
		t.Fatalf("ParseClaudeCodeRateLimit(nil) = %+v, want nil", rateLimit)
	}
}

func TestParseCodexRateLimit(t *testing.T) {
	rateLimit, err := ParseCodexRateLimit([]byte(`{
		"limitId":"codex",
		"primary":{"usedPercent":1.0,"windowDurationMins":300,"resetsAt":1775050232},
		"secondary":{"usedPercent":0.125,"windowDurationMins":10080,"resetsAt":1775637032},
		"planType":"pro"
	}`))
	if err != nil {
		t.Fatalf("ParseCodexRateLimit returned error: %v", err)
	}
	if rateLimit == nil || rateLimit.Codex == nil {
		t.Fatalf("ParseCodexRateLimit returned %+v", rateLimit)
	}
	if rateLimit.Provider != CLIRateLimitProviderCodex {
		t.Fatalf("provider = %q", rateLimit.Provider)
	}
	if rateLimit.Codex.LimitID != "codex" || rateLimit.Codex.PlanType != "pro" {
		t.Fatalf("Codex payload = %+v", rateLimit.Codex)
	}
	if rateLimit.Codex.Primary == nil || rateLimit.Codex.Primary.UsedPercent == nil || *rateLimit.Codex.Primary.UsedPercent != 1.0 {
		t.Fatalf("primary window = %+v", rateLimit.Codex.Primary)
	}
	if rateLimit.Codex.Primary.ResetsAt == nil || !rateLimit.Codex.Primary.ResetsAt.Equal(time.Unix(1775050232, 0).UTC()) {
		t.Fatalf("primary resets_at = %+v", rateLimit.Codex.Primary.ResetsAt)
	}
}

func TestParseGeminiCLIRateLimit(t *testing.T) {
	rateLimit, err := ParseGeminiCLIRateLimit([]byte(`{
		"authType":"oauth-personal",
		"remaining":12,
		"limit":50,
		"resetTime":"2026-04-02T10:02:55Z",
		"buckets":[
			{
				"modelId":"gemini-2.5-pro",
				"tokenType":"REQUESTS",
				"remainingAmount":"12",
				"remainingFraction":0.24,
				"resetTime":"2026-04-02T10:02:55Z"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseGeminiCLIRateLimit returned error: %v", err)
	}
	if rateLimit == nil || rateLimit.Gemini == nil {
		t.Fatalf("ParseGeminiCLIRateLimit returned %+v", rateLimit)
	}
	if rateLimit.Provider != CLIRateLimitProviderGemini {
		t.Fatalf("provider = %q", rateLimit.Provider)
	}
	if rateLimit.Gemini.AuthType != "oauth-personal" {
		t.Fatalf("auth_type = %q", rateLimit.Gemini.AuthType)
	}
	if rateLimit.Gemini.Remaining == nil || *rateLimit.Gemini.Remaining != 12 {
		t.Fatalf("remaining = %+v", rateLimit.Gemini.Remaining)
	}
	if len(rateLimit.Gemini.Buckets) != 1 || rateLimit.Gemini.Buckets[0].ModelID != "gemini-2.5-pro" {
		t.Fatalf("buckets = %+v", rateLimit.Gemini.Buckets)
	}
}

func TestProbeGeminiCLIRateLimitParsesProcessOutput(t *testing.T) {
	tempDir := t.TempDir()
	geminiPath := filepath.Join(tempDir, "gemini")
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("LookPath returned error: %v", err)
	}
	if err := os.Symlink(nodePath, geminiPath); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	manager := &stubRateLimitProcessManager{
		process: &stubRateLimitProcess{
			stdout: io.NopCloser(bytes.NewReader([]byte(`{"authType":"oauth-personal","remaining":3,"limit":10,"resetTime":"2026-04-02T10:02:55Z","buckets":[{"modelId":"gemini-2.5-pro","tokenType":"REQUESTS","remainingFraction":0.3,"resetTime":"2026-04-02T10:02:55Z"}]}`))),
			stderr: io.NopCloser(strings.NewReader("debug log\n")),
		},
	}

	rateLimit, observedAt, err := ProbeGeminiCLIRateLimit(
		context.Background(),
		manager,
		MustParseAgentCLICommand("gemini"),
		nil,
		nil,
		"gemini-2.5-pro",
	)
	if err != nil {
		t.Fatalf("ProbeGeminiCLIRateLimit returned error: %v", err)
	}
	if manager.spec.Command != MustParseAgentCLICommand("node") {
		t.Fatalf("probe command = %q", manager.spec.Command)
	}
	if rateLimit == nil || rateLimit.Gemini == nil {
		t.Fatalf("ProbeGeminiCLIRateLimit returned %+v", rateLimit)
	}
	if rateLimit.Gemini.Remaining == nil || *rateLimit.Gemini.Remaining != 3 {
		t.Fatalf("remaining = %+v", rateLimit.Gemini.Remaining)
	}
	if observedAt == nil {
		t.Fatal("observedAt = nil")
	}
}

type stubRateLimitProcessManager struct {
	process AgentCLIProcess
	spec    AgentCLIProcessSpec
}

func (m *stubRateLimitProcessManager) Start(_ context.Context, spec AgentCLIProcessSpec) (AgentCLIProcess, error) {
	m.spec = spec
	return m.process, nil
}

type stubRateLimitProcess struct {
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	waitErr error
}

func (p *stubRateLimitProcess) PID() int              { return 31337 }
func (p *stubRateLimitProcess) Stdin() io.WriteCloser { return rateLimitNopWriteCloser{} }
func (p *stubRateLimitProcess) Stdout() io.ReadCloser { return p.stdout }
func (p *stubRateLimitProcess) Stderr() io.ReadCloser { return p.stderr }
func (p *stubRateLimitProcess) Wait() error           { return p.waitErr }
func (p *stubRateLimitProcess) Stop(context.Context) error {
	return nil
}

type rateLimitNopWriteCloser struct{}

func (rateLimitNopWriteCloser) Write(data []byte) (int, error) { return len(data), nil }
func (rateLimitNopWriteCloser) Close() error                   { return nil }
