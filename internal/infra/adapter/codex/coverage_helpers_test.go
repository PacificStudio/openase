package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestProtocolAndHelperCoverage(t *testing.T) {
	t.Parallel()

	numeric := newNumericRequestID(42)
	rawNumeric, err := numeric.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(rawNumeric) != "42" || numeric.String() != "42" {
		t.Fatalf("numeric request ID = (%s, %q)", string(rawNumeric), numeric.String())
	}

	parsed, err := parseRequestID(json.RawMessage(` "req-1" `))
	if err != nil {
		t.Fatalf("parseRequestID() error = %v", err)
	}
	if parsed.String() != `"req-1"` {
		t.Fatalf("parseRequestID() = %q", parsed.String())
	}
	if _, err := parseRequestID(json.RawMessage(`true`)); err == nil || !strings.Contains(err.Error(), "string or number") {
		t.Fatalf("parseRequestID(bool) error = %v", err)
	}
	if _, err := (RequestID{}).MarshalJSON(); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("empty MarshalJSON() error = %v", err)
	}

	if err := (jsonRPCMessage{JSONRPC: "1.0"}).validate(); err == nil || !strings.Contains(err.Error(), "unsupported jsonrpc version") {
		t.Fatalf("validate(version) error = %v", err)
	}
	if err := (jsonRPCMessage{JSONRPC: jsonRPCVersion, Method: methodInitialize, Result: mustMarshalJSON(map[string]any{})}).validate(); err == nil || !strings.Contains(err.Error(), "must not mix method") {
		t.Fatalf("validate(method/result) error = %v", err)
	}
	if err := (jsonRPCMessage{JSONRPC: jsonRPCVersion, Result: mustMarshalJSON(map[string]any{})}).validate(); err == nil || !strings.Contains(err.Error(), "must include an id") {
		t.Fatalf("validate(missing id) error = %v", err)
	}
	if err := (jsonRPCMessage{JSONRPC: jsonRPCVersion, ID: mustMarshalJSON(1)}).validate(); err == nil || !strings.Contains(err.Error(), "either result or error") {
		t.Fatalf("validate(missing result/error) error = %v", err)
	}
	if err := (jsonRPCMessage{JSONRPC: jsonRPCVersion, ID: mustMarshalJSON(1), Result: mustMarshalJSON(map[string]any{"ok": true})}).validate(); err != nil {
		t.Fatalf("validate(valid response) error = %v", err)
	}

	if got, ok := approvalOptionLabel([]struct {
		Label string `json:"label"`
	}{
		{Label: "Deny"},
		{Label: "Allow execution"},
	}); !ok || got != "Allow execution" {
		t.Fatalf("approvalOptionLabel() = (%q, %v)", got, ok)
	}
	if got, ok := approvalOptionLabel([]struct {
		Label string `json:"label"`
	}{{Label: "Deny"}}); ok || got != "" {
		t.Fatalf("approvalOptionLabel(no match) = (%q, %v)", got, ok)
	}

	if got := optionalString("  "); got != nil {
		t.Fatalf("optionalString(blank) = %v", got)
	}
	if got := optionalAbsolutePath("/tmp/openase"); got == nil || *got != "/tmp/openase" {
		t.Fatalf("optionalAbsolutePath(valid) = %v", got)
	}
	if got := optionalAbsolutePath("relative/path"); got != nil {
		t.Fatalf("optionalAbsolutePath(relative) = %v", got)
	}
	if !approvalPolicyIsNever(" Never ") || approvalPolicyIsNever("auto") {
		t.Fatal("approvalPolicyIsNever() mismatch")
	}

	cloned := cloneJSONCompatibleValue(map[string]any{"nested": []string{"a", "b"}})
	decodedMap, ok := cloned.(map[string]any)
	if !ok || len(decodedMap) != 1 {
		t.Fatalf("cloneJSONCompatibleValue(map) = %#v", cloned)
	}
	fallback := cloneJSONCompatibleValue(map[string]any{"bad": make(chan int)})
	if _, ok := fallback.(map[string]any); !ok {
		t.Fatalf("cloneJSONCompatibleValue(unmarshalable) = %#v", fallback)
	}

	var params struct {
		Value string `json:"value"`
	}
	if err := decodeParams(json.RawMessage(`{"value":"ok"}`), &params); err != nil || params.Value != "ok" {
		t.Fatalf("decodeParams() = (%+v, %v)", params, err)
	}
	if err := decodeParams(json.RawMessage(` `), &params); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("decodeParams(empty) error = %v", err)
	}

	if got := coalesceError(nil, "fallback"); got != "fallback" {
		t.Fatalf("coalesceError(nil) = %q", got)
	}
	if got := coalesceError(errors.New("boom"), "fallback"); got != "boom" {
		t.Fatalf("coalesceError(err) = %q", got)
	}

	if turnErrorFromWire(nil) != nil {
		t.Fatal("turnErrorFromWire(nil) != nil")
	}
	wireErr := turnErrorFromWire(&wireTurnError{Message: "bad", AdditionalDetails: "details"})
	if wireErr == nil || wireErr.Message != "bad" || wireErr.AdditionalDetails != "details" {
		t.Fatalf("turnErrorFromWire() = %+v", wireErr)
	}

	session := &Session{
		defaultTurnWorkingDirectory: "/tmp/default",
		defaultTurnTitle:            "Default",
		defaultApprovalPolicy:       map[string]any{"mode": "auto"},
		defaultSandboxPolicy:        []string{"workspace-write"},
	}
	merged := mergeTurnConfig(session, TurnConfig{
		WorkingDirectory: " /tmp/override ",
		Title:            " Override ",
		ApprovalPolicy:   map[string]any{"mode": "never"},
	})
	if merged.WorkingDirectory != "/tmp/override" || merged.Title != "Override" {
		t.Fatalf("mergeTurnConfig() = %+v", merged)
	}
	if policy, ok := merged.ApprovalPolicy.(map[string]any); !ok || policy["mode"] != "never" {
		t.Fatalf("mergeTurnConfig().ApprovalPolicy = %#v", merged.ApprovalPolicy)
	}
	if sandbox, ok := merged.SandboxPolicy.([]any); !ok || len(sandbox) != 1 || sandbox[0] != "workspace-write" {
		t.Fatalf("mergeTurnConfig().SandboxPolicy = %#v", merged.SandboxPolicy)
	}

	process := newFakeProcess()
	stopped := newSession(process)
	stopped.doneErr = errors.New("session failed")
	close(stopped.done)
	if err := stopped.stopWithTimeout(); err != nil {
		t.Fatalf("stopWithTimeout() error = %v", err)
	}
	if err := stopped.sessionError(); err == nil || err.Error() != "session failed" {
		t.Fatalf("sessionError() = %v", err)
	}
	stopped.stderr.WriteString(" stderr line ")
	if got := stopped.stderrSuffix(); got != ": stderr line" {
		t.Fatalf("stderrSuffix() = %q", got)
	}
}

func TestSessionApprovalAndUserInputResponses(t *testing.T) {
	t.Parallel()

	requestID, err := parseRequestID(json.RawMessage(`"approval-1"`))
	if err != nil {
		t.Fatalf("parseRequestID() error = %v", err)
	}

	denyBuffer := bytes.NewBuffer(nil)
	denySession := &Session{encoder: json.NewEncoder(denyBuffer)}
	if err := denySession.respondApproval(requestID, "acceptForSession"); err != nil {
		t.Fatalf("respondApproval(disabled) error = %v", err)
	}
	var denyMessage jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(denyBuffer.Bytes()), &denyMessage); err != nil {
		t.Fatalf("Unmarshal disabled approval message: %v", err)
	}
	if denyMessage.Error == nil || !strings.Contains(denyMessage.Error.Message, "interactive approval is not supported") {
		t.Fatalf("disabled approval message = %+v", denyMessage)
	}

	approveBuffer := bytes.NewBuffer(nil)
	approveSession := &Session{encoder: json.NewEncoder(approveBuffer), autoApproveRequests: true}
	if err := approveSession.respondApproval(requestID, "approved_for_session"); err != nil {
		t.Fatalf("respondApproval(enabled) error = %v", err)
	}
	var approveMessage jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(approveBuffer.Bytes()), &approveMessage); err != nil {
		t.Fatalf("Unmarshal enabled approval message: %v", err)
	}
	var approvePayload map[string]string
	if err := json.Unmarshal(approveMessage.Result, &approvePayload); err != nil {
		t.Fatalf("Unmarshal enabled approval payload: %v", err)
	}
	if approvePayload["decision"] != "approved_for_session" {
		t.Fatalf("approval payload = %+v", approvePayload)
	}

	inputBuffer := bytes.NewBuffer(nil)
	inputSession := &Session{encoder: json.NewEncoder(inputBuffer), autoApproveRequests: true}
	inputPayloadMap := map[string]any{
		"questions": []any{
			map[string]any{"id": "approval", "options": []any{map[string]any{"label": "Deny"}, map[string]any{"label": "Approve Once"}}},
			map[string]any{"id": "fallback", "options": []any{map[string]any{"label": "Deny"}}},
		},
	}
	if err := inputSession.RespondUserInput(context.Background(), UserInputRequest{RequestID: requestID}, defaultToolRequestUserInputAnswers(inputPayloadMap)); err != nil {
		t.Fatalf("RespondUserInput() error = %v", err)
	}
	var inputMessage jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(inputBuffer.Bytes()), &inputMessage); err != nil {
		t.Fatalf("Unmarshal requestUserInput message: %v", err)
	}
	var inputPayload struct {
		Answers map[string]struct {
			Answers []string `json:"answers"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(inputMessage.Result, &inputPayload); err != nil {
		t.Fatalf("Unmarshal requestUserInput payload: %v", err)
	}
	if inputPayload.Answers["approval"].Answers[0] != "Approve Once" {
		t.Fatalf("approval answer = %+v", inputPayload.Answers["approval"])
	}
	if inputPayload.Answers["fallback"].Answers[0] != defaultToolInputAnswer {
		t.Fatalf("fallback answer = %+v", inputPayload.Answers["fallback"])
	}

	errorBuffer := bytes.NewBuffer(nil)
	errorSession := &Session{encoder: json.NewEncoder(errorBuffer)}
	if err := errorSession.RespondUserInput(context.Background(), UserInputRequest{RequestID: requestID}, map[string]any{}); err != nil {
		t.Fatalf("RespondUserInput(empty answers) error = %v", err)
	}
	var errorMessage jsonRPCMessage
	if err := json.Unmarshal(bytes.TrimSpace(errorBuffer.Bytes()), &errorMessage); err != nil {
		t.Fatalf("Unmarshal empty-ID requestUserInput message: %v", err)
	}
	if errorMessage.Error == nil || !strings.Contains(errorMessage.Error.Message, "requires at least one question") {
		t.Fatalf("empty-ID requestUserInput message = %+v", errorMessage)
	}
}
