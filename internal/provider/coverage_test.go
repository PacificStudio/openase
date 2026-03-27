package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testContextKey struct{}

func TestProviderAgentCLIHelpers(t *testing.T) {
	t.Parallel()

	command, err := ParseAgentCLICommand(" codex ")
	if err != nil || command != "codex" {
		t.Fatalf("ParseAgentCLICommand() = %q, %v", command, err)
	}
	if command.String() != "codex" {
		t.Fatalf("AgentCLICommand.String() = %q", command.String())
	}
	if got := MustParseAgentCLICommand("claude"); got != "claude" {
		t.Fatalf("MustParseAgentCLICommand() = %q", got)
	}
	assertPanics(t, func() { MustParseAgentCLICommand(" ") })

	workingDirectory := MustParseAbsolutePath("/tmp/openase")
	spec, err := NewAgentCLIProcessSpec(
		command,
		[]string{"run", "--json"},
		&workingDirectory,
		[]string{"OPENASE=1", "TOKEN=value"},
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec() error = %v", err)
	}
	if spec.Command != command || spec.WorkingDirectory == nil || spec.Args[0] != "run" || spec.Environment[1] != "TOKEN=value" {
		t.Fatalf("NewAgentCLIProcessSpec() = %+v", spec)
	}

	tests := []struct {
		name string
		arg  string
	}{
		{name: "empty", arg: ""},
		{name: "missing equals", arg: "TOKEN"},
		{name: "blank key", arg: " =value"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateProcessEnvironmentEntry(tt.arg); err == nil {
				t.Fatalf("validateProcessEnvironmentEntry(%q) expected error", tt.arg)
			}
		})
	}
	if _, err := NewAgentCLIProcessSpec("", nil, nil, nil); err == nil {
		t.Fatal("NewAgentCLIProcessSpec() expected empty command error")
	}
	emptyWorkingDirectory := AbsolutePath("")
	if _, err := NewAgentCLIProcessSpec(command, nil, &emptyWorkingDirectory, nil); err == nil {
		t.Fatal("NewAgentCLIProcessSpec() expected empty working directory error")
	}
}

func TestProviderClaudeCodeHelpers(t *testing.T) {
	t.Parallel()

	sessionID, err := ParseClaudeCodeSessionID(" session-1 ")
	if err != nil || sessionID != "session-1" {
		t.Fatalf("ParseClaudeCodeSessionID() = %q, %v", sessionID, err)
	}
	if got := MustParseClaudeCodeSessionID("session-2"); got.String() != "session-2" {
		t.Fatalf("MustParseClaudeCodeSessionID() = %q", got)
	}
	assertPanics(t, func() { MustParseClaudeCodeSessionID(" ") })

	turn, err := NewClaudeCodeTurnInput(" ship it ")
	if err != nil || turn.Prompt != "ship it" {
		t.Fatalf("NewClaudeCodeTurnInput() = %+v, %v", turn, err)
	}
	if _, err := NewClaudeCodeTurnInput(" "); err == nil {
		t.Fatal("NewClaudeCodeTurnInput() expected error")
	}

	command := MustParseAgentCLICommand("codex")
	workingDirectory := MustParseAbsolutePath("/tmp/openase")
	maxTurns := 4
	maxBudget := 12.5
	resumeID := MustParseClaudeCodeSessionID("resume-1")
	spec, err := NewClaudeCodeSessionSpec(
		command,
		[]string{"run"},
		&workingDirectory,
		[]string{"OPENASE=1"},
		[]string{"read", " write "},
		"  system prompt  ",
		&maxTurns,
		&maxBudget,
		&resumeID,
		true,
	)
	if err != nil {
		t.Fatalf("NewClaudeCodeSessionSpec() error = %v", err)
	}
	if spec.Command != command || len(spec.AllowedTools) != 2 || spec.AllowedTools[1] != "write" || spec.AppendSystemPrompt != "system prompt" {
		t.Fatalf("NewClaudeCodeSessionSpec() = %+v", spec)
	}
	if spec.MaxTurns == &maxTurns || spec.MaxBudgetUSD == &maxBudget || spec.ResumeSessionID == &resumeID {
		t.Fatal("NewClaudeCodeSessionSpec() should clone pointer inputs")
	}

	if _, err := NewClaudeCodeSessionSpec("", nil, nil, nil, nil, "", nil, nil, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected empty command error")
	}
	emptyWorkingDirectory := AbsolutePath("")
	if _, err := NewClaudeCodeSessionSpec(command, nil, &emptyWorkingDirectory, nil, nil, "", nil, nil, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected empty working directory error")
	}
	if _, err := NewClaudeCodeSessionSpec(command, nil, nil, []string{"BROKEN"}, nil, "", nil, nil, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected environment error")
	}
	if _, err := NewClaudeCodeSessionSpec(command, nil, nil, nil, []string{"", "read"}, "", nil, nil, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected allowed tools error")
	}
	zeroTurns := 0
	if _, err := NewClaudeCodeSessionSpec(command, nil, nil, nil, nil, "", &zeroTurns, nil, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected max turns error")
	}
	zeroBudget := 0.0
	if _, err := NewClaudeCodeSessionSpec(command, nil, nil, nil, nil, "", nil, &zeroBudget, nil, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected max budget error")
	}
	emptyResumeID := ClaudeCodeSessionID("")
	if _, err := NewClaudeCodeSessionSpec(command, nil, nil, nil, nil, "", nil, nil, &emptyResumeID, false); err == nil {
		t.Fatal("NewClaudeCodeSessionSpec() expected empty resume id error")
	}
}

func TestProviderEnvironmentHelpers(t *testing.T) {
	t.Parallel()

	authConfig := map[string]any{
		"token":      "abc",
		"max-retry":  3,
		"enabled":    true,
		"threshold":  json.Number("12.5"),
		"bad object": map[string]string{"nested": "x"},
		"":           "skip",
	}
	env := AuthConfigEnvironment(authConfig)
	if want := []string{"ENABLED=true", "MAX_RETRY=3", "THRESHOLD=12.5", "TOKEN=abc"}; strings.Join(env, ",") != strings.Join(want, ",") {
		t.Fatalf("AuthConfigEnvironment() = %v, want %v", env, want)
	}
	if got, ok := LookupEnvironmentValue([]string{"TOKEN=old", "TOKEN=new"}, " token "); !ok || got != "new" {
		t.Fatalf("LookupEnvironmentValue() = %q, %v", got, ok)
	}
	if _, ok := LookupEnvironmentValue([]string{"TOKEN=value"}, " "); ok {
		t.Fatal("LookupEnvironmentValue() expected false for blank key")
	}
	if got, ok := stringifyEnvValue(1.5); !ok || got != "1.5" {
		t.Fatalf("stringifyEnvValue(float64) = %q, %v", got, ok)
	}
	if _, ok := stringifyEnvValue(struct{}{}); ok {
		t.Fatal("stringifyEnvValue(struct) expected false")
	}
	if got := normalizeEnvKey("  openase.token-value "); got != "OPENASE_TOKEN_VALUE" {
		t.Fatalf("normalizeEnvKey() = %q", got)
	}
	if got := normalizeEnvKey(" "); got != "" {
		t.Fatalf("normalizeEnvKey(blank) = %q", got)
	}
}

func TestProviderServiceAndPathHelpers(t *testing.T) {
	t.Parallel()

	name, err := ParseServiceName(" openase.service ")
	if err != nil || name != "openase.service" {
		t.Fatalf("ParseServiceName() = %q, %v", name, err)
	}
	if got := MustParseServiceName("openase-service"); got.String() != "openase-service" {
		t.Fatalf("MustParseServiceName() = %q", got)
	}
	assertPanics(t, func() { MustParseServiceName(" ") })

	absPath, err := ParseAbsolutePath(filepath.Join("/", "tmp", "openase"))
	if err != nil || absPath.String() != filepath.Join("/", "tmp", "openase") {
		t.Fatalf("ParseAbsolutePath() = %q, %v", absPath, err)
	}
	if got := MustParseAbsolutePath("/tmp/../tmp/openase"); got.String() != "/tmp/openase" {
		t.Fatalf("MustParseAbsolutePath() = %q", got)
	}
	assertPanics(t, func() { MustParseAbsolutePath("relative") })

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	logsOptions, err := NewUserServiceLogsOptions(20, true, stdout, stderr)
	if err != nil || logsOptions.Lines != 20 || !logsOptions.Follow {
		t.Fatalf("NewUserServiceLogsOptions() = %+v, %v", logsOptions, err)
	}
	if _, err := NewUserServiceLogsOptions(0, false, stdout, stderr); err == nil {
		t.Fatal("NewUserServiceLogsOptions() expected line-count error")
	}
	if _, err := NewUserServiceLogsOptions(1, false, nil, stderr); err == nil {
		t.Fatal("NewUserServiceLogsOptions() expected stdout error")
	}
	if _, err := NewUserServiceLogsOptions(1, false, stdout, nil); err == nil {
		t.Fatal("NewUserServiceLogsOptions() expected stderr error")
	}

	spec, err := NewUserServiceInstallSpec(
		name,
		" OpenASE runtime ",
		MustParseAbsolutePath("/usr/local/bin/openase"),
		[]string{"serve"},
		MustParseAbsolutePath("/srv/openase"),
		MustParseAbsolutePath("/etc/openase.env"),
		MustParseAbsolutePath("/var/log/openase/stdout.log"),
		MustParseAbsolutePath("/var/log/openase/stderr.log"),
	)
	if err != nil {
		t.Fatalf("NewUserServiceInstallSpec() error = %v", err)
	}
	if spec.Description != "OpenASE runtime" || spec.Arguments[0] != "serve" {
		t.Fatalf("NewUserServiceInstallSpec() = %+v", spec)
	}
	if _, err := NewUserServiceInstallSpec("", "desc", "/bin/openase", nil, "/srv", "/env", "/out", "/err"); err == nil {
		t.Fatal("NewUserServiceInstallSpec() expected empty name error")
	}
	if _, err := NewUserServiceInstallSpec(name, " ", "/bin/openase", nil, "/srv", "/env", "/out", "/err"); err == nil {
		t.Fatal("NewUserServiceInstallSpec() expected description error")
	}
}

func TestProviderTraceAndMetricHelpers(t *testing.T) {
	t.Parallel()

	traceProvider := NewNoopTraceProvider()
	ctx := context.WithValue(context.Background(), testContextKey{}, "value")
	header := http.Header{}
	if got := traceProvider.ExtractHTTPContext(ctx, header); got != ctx {
		t.Fatalf("ExtractHTTPContext() = %#v, want original context", got)
	}
	traceProvider.InjectHTTPHeaders(ctx, header)

	spanCtx, span := traceProvider.StartSpan(ctx, "runtime.tick", WithSpanKind(SpanKindServer), WithSpanAttributes(StringAttribute("mode", "serve"), IntAttribute("tickets", 3), BoolAttribute("healthy", true)))
	if spanCtx != ctx {
		t.Fatalf("StartSpan() context = %#v, want original context", spanCtx)
	}
	span.RecordError(errors.New("boom"))
	span.SetAttributes(StringAttribute("extra", "value"))
	span.SetStatus(SpanStatusOK, "")
	span.End()
	if span.TraceID() != "" || span.SpanID() != "" {
		t.Fatalf("noop span ids = %q/%q, want empty", span.TraceID(), span.SpanID())
	}
	if err := traceProvider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	kind, attrs := ResolveSpanStartOptions(nil, WithSpanKind(SpanKindClient), WithSpanAttributes(StringAttribute("mode", "serve")))
	if kind != SpanKindClient || len(attrs) != 1 || attrs[0].StringValue != "serve" {
		t.Fatalf("ResolveSpanStartOptions() = %v, %+v", kind, attrs)
	}
	defaultKind, defaultAttrs := ResolveSpanStartOptions()
	if defaultKind != SpanKindInternal || len(defaultAttrs) != 0 {
		t.Fatalf("ResolveSpanStartOptions() defaults = %v, %+v", defaultKind, defaultAttrs)
	}

	metricsProvider := NewNoopMetricsProvider()
	metricsProvider.Counter("openase.counter", Tags{"mode": "serve"}).Add(1)
	metricsProvider.Histogram("openase.histogram", nil).Record(1.5)
	metricsProvider.Gauge("openase.gauge", nil).Set(2)
}

func TestProviderEventHelpers(t *testing.T) {
	t.Parallel()

	topic, err := ParseTopic("runtime.events")
	if err != nil || topic.String() != "runtime.events" {
		t.Fatalf("ParseTopic() = %q, %v", topic, err)
	}
	eventType, err := ParseEventType("orchestrator.tick")
	if err != nil || eventType.String() != "orchestrator.tick" {
		t.Fatalf("ParseEventType() = %q, %v", eventType, err)
	}
	if got := MustParseTopic("runtime.events"); got != "runtime.events" {
		t.Fatalf("MustParseTopic() = %q", got)
	}
	if got := MustParseEventType("runtime.started"); got != "runtime.started" {
		t.Fatalf("MustParseEventType() = %q", got)
	}
	assertPanics(t, func() { MustParseTopic(" ") })
	assertPanics(t, func() { MustParseEventType(" ") })

	if _, err := ParseTopic(" "); err == nil {
		t.Fatal("ParseTopic() expected blank error")
	}
	if _, err := ParseEventType("bad topic"); err == nil {
		t.Fatal("ParseEventType() expected token error")
	}

	event, err := NewEvent(topic, eventType, json.RawMessage(`{"status":"ok"}`), mustTime("2026-03-27T10:00:00+02:00"))
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if event.PublishedAt.Location() != time.UTC {
		t.Fatalf("NewEvent() published_at location = %v, want UTC", event.PublishedAt.Location())
	}
	if _, err := NewEvent("", eventType, nil, mustTime("2026-03-27T10:00:00Z")); err == nil {
		t.Fatal("NewEvent() expected topic error")
	}
	if _, err := NewEvent(topic, "", nil, mustTime("2026-03-27T10:00:00Z")); err == nil {
		t.Fatal("NewEvent() expected event type error")
	}
	if _, err := NewEvent(topic, eventType, nil, time.Time{}); err == nil {
		t.Fatal("NewEvent() expected published_at error")
	}
	if _, err := NewJSONEvent(topic, eventType, func() {}, mustTime("2026-03-27T10:00:00Z")); err == nil {
		t.Fatal("NewJSONEvent() expected marshal error")
	}
}

func mustTime(raw string) time.Time {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	return value
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

var _ AgentCLIProcess = (*testAgentCLIProcess)(nil)

type testAgentCLIProcess struct{}

func (testAgentCLIProcess) PID() int                   { return 0 }
func (testAgentCLIProcess) Stdin() io.WriteCloser      { return nopWriteCloser{Writer: io.Discard} }
func (testAgentCLIProcess) Stdout() io.ReadCloser      { return io.NopCloser(strings.NewReader("")) }
func (testAgentCLIProcess) Stderr() io.ReadCloser      { return io.NopCloser(strings.NewReader("")) }
func (testAgentCLIProcess) Wait() error                { return nil }
func (testAgentCLIProcess) Stop(context.Context) error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
