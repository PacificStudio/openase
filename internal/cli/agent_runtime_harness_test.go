package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	runtimeHarnessOrgID        = "11111111-1111-1111-1111-111111111111"
	runtimeHarnessProjectID    = "22222222-2222-2222-2222-222222222222"
	runtimeHarnessTicketID     = "33333333-3333-3333-3333-333333333333"
	runtimeHarnessWorkflowID   = "44444444-4444-4444-4444-444444444444"
	runtimeHarnessAgentID      = "55555555-5555-5555-5555-555555555555"
	runtimeHarnessRunID        = "66666666-6666-6666-6666-666666666666"
	runtimeHarnessRepoID       = "77777777-7777-7777-7777-777777777777"
	runtimeHarnessScopeID      = "88888888-8888-8888-8888-888888888888"
	runtimeHarnessRuleID       = "99999999-9999-9999-9999-999999999999"
	runtimeHarnessStatusID     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	runtimeHarnessCommentID    = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	runtimeHarnessDependencyID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	runtimeHarnessLinkID       = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	runtimeHarnessThreadID     = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	runtimeHarnessJobID        = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	runtimeHarnessSkillID      = "12121212-1212-1212-1212-121212121212"
)

var nonAgentRuntimeTopLevelCommands = map[string]string{
	"all-in-one":        "local service lifecycle command, not an agent runtime API semantic",
	"api":               "raw passthrough command, not a stable agent runtime semantic leaf",
	"auth":              "human/browser bootstrap and session management are outside agent runtime semantics",
	"doctor":            "local environment diagnostics are outside agent runtime semantics",
	"down":              "local service lifecycle command, not an agent runtime API semantic",
	"issue-agent-token": "token issuance is an admin/control-plane action, not a runtime agent command",
	"logs":              "local service lifecycle command, not an agent runtime API semantic",
	"machine-agent":     "machine-side helper commands are not project agent runtime semantics",
	"openapi":           "OpenAPI inspection is a documentation utility, not a runtime agent command",
	"orchestrate":       "local service lifecycle command, not an agent runtime API semantic",
	"serve":             "local service lifecycle command, not an agent runtime API semantic",
	"setup":             "local bootstrap command, not an agent runtime API semantic",
	"up":                "local service lifecycle command, not an agent runtime API semantic",
	"version":           "version inspection is a utility command, not a runtime agent command",
}

type agentRuntimeLeaf struct {
	path      []string
	command   *cobra.Command
	method    string
	humanPath string
	contract  openAPICommandContract
	hasSpec   bool
}

func TestAgentRuntimeLeafHarnessCoversRequiredNamespaces(t *testing.T) {
	leaves := collectAgentRuntimeLeaves(t)
	if len(leaves) == 0 {
		t.Fatal("expected at least one agent-runtime leaf command")
	}

	covered := make(map[string]struct{}, len(leaves))
	for _, leaf := range leaves {
		covered[strings.Join(leaf.path, " ")] = struct{}{}
	}

	for _, wantPrefix := range []string{
		"activity ",
		"agent ",
		"notification-rule ",
		"project ",
		"repo ",
		"scheduled-job ",
		"skill ",
		"status ",
		"ticket ",
		"workflow ",
	} {
		if hasPathPrefix(covered, wantPrefix) {
			continue
		}
		t.Fatalf("agent runtime harness is missing namespace coverage for %q", strings.TrimSpace(wantPrefix))
	}
}

func TestAgentRuntimeLeafSelectionExcludesNonRuntimeCommands(t *testing.T) {
	root := NewRootCommand("dev")

	tests := []struct {
		path       []string
		wantReason string
	}{
		{
			path:       []string{"serve"},
			wantReason: nonAgentRuntimeTopLevelCommands["serve"],
		},
		{
			path:       []string{"auth", "bootstrap", "login"},
			wantReason: nonAgentRuntimeTopLevelCommands["auth"],
		},
		{
			path:       []string{"machine", "list"},
			wantReason: "operation has no /api/v1/platform counterpart",
		},
		{
			path:       []string{"provider", "get"},
			wantReason: "operation has no /api/v1/platform counterpart",
		},
		{
			path:       []string{"channel", "test"},
			wantReason: "operation has no /api/v1/platform counterpart",
		},
		{
			path:       []string{"watch", "events"},
			wantReason: "operation has no /api/v1/platform counterpart",
		},
		{
			path:       []string{"issue-agent-token"},
			wantReason: nonAgentRuntimeTopLevelCommands["issue-agent-token"],
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, "_"), func(t *testing.T) {
			command, _, err := root.Find(tt.path)
			if err != nil {
				t.Fatalf("Find(%v) returned error: %v", tt.path, err)
			}
			if command == nil {
				t.Fatalf("expected command %v", tt.path)
			}

			_, ok, reason := classifyAgentRuntimeLeaf(tt.path, command, nil)
			if ok {
				t.Fatalf("expected %v to be excluded from agent runtime harness", tt.path)
			}
			if reason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestAgentRuntimeLeafCommandsUsePlatformRoutesInAgentRuntime(t *testing.T) {
	runAgentRuntimeLeafRouteHarness(t, "/api/v1/platform")
}

func TestAgentRuntimeLeafCommandsUseHumanRoutesWithHumanBase(t *testing.T) {
	runAgentRuntimeLeafRouteHarness(t, "/api/v1")
}

func runAgentRuntimeLeafRouteHarness(t *testing.T, basePath string) {
	t.Helper()

	for _, leaf := range collectAgentRuntimeLeaves(t) {
		t.Run(strings.Join(leaf.path, "_"), func(t *testing.T) {
			t.Setenv("OPENASE_AGENT_TOKEN", "ase_agent_test")
			t.Setenv("OPENASE_ORG_ID", runtimeHarnessOrgID)
			t.Setenv("OPENASE_PROJECT_ID", runtimeHarnessProjectID)
			t.Setenv("OPENASE_TICKET_ID", runtimeHarnessTicketID)

			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.RequestURI())
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			t.Setenv("OPENASE_API_URL", server.URL+basePath)

			root := NewRootCommand("dev")
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stdout)
			root.SetArgs(runtimeLeafArgs(t, leaf))

			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext(%v) returned error: %v\noutput: %s", leaf.path, err, stdout.String())
			}
			if len(requests) != 1 {
				t.Fatalf("request count = %d, want 1 (%v)", len(requests), requests)
			}

			expectedPath := runtimeExpectedRequestPath(leaf, basePath)
			got := requests[0]
			want := leaf.method + " " + expectedPath
			if got != want {
				t.Fatalf("request = %q, want %q", got, want)
			}
		})
	}
}

func collectAgentRuntimeLeaves(t *testing.T) []agentRuntimeLeaf {
	t.Helper()

	contracts, err := loadOpenAPICommandContracts()
	if err != nil {
		t.Fatalf("load OpenAPI command contracts: %v", err)
	}

	root := NewRootCommand("dev")
	leaves := make([]agentRuntimeLeaf, 0, 64)
	walkCLILeaves(root, []string{}, func(path []string, command *cobra.Command) {
		leaf, ok, _ := classifyAgentRuntimeLeaf(path, command, contracts)
		if ok {
			leaves = append(leaves, leaf)
		}
	})

	slices.SortFunc(leaves, func(left, right agentRuntimeLeaf) int {
		return strings.Compare(strings.Join(left.path, " "), strings.Join(right.path, " "))
	})
	return leaves
}

func classifyAgentRuntimeLeaf(path []string, command *cobra.Command, contracts map[string]openAPICommandContract) (agentRuntimeLeaf, bool, string) {
	if len(path) == 0 || command == nil {
		return agentRuntimeLeaf{}, false, "missing leaf metadata"
	}
	if rationale, excluded := nonAgentRuntimeTopLevelCommands[path[0]]; excluded {
		return agentRuntimeLeaf{}, false, rationale
	}

	key, ok := cliCommandAPICoverageKey(command)
	if !ok {
		return agentRuntimeLeaf{}, false, "command has no typed API coverage annotation"
	}
	parts := strings.SplitN(key, " ", 2)
	if len(parts) != 2 {
		return agentRuntimeLeaf{}, false, "command coverage annotation is malformed"
	}
	method := strings.ToUpper(strings.TrimSpace(parts[0]))
	humanPath := strings.TrimSpace(parts[1])
	if !httpapi.HasAgentPlatformRoute(method, humanPath) {
		return agentRuntimeLeaf{}, false, "operation has no /api/v1/platform counterpart"
	}

	leaf := agentRuntimeLeaf{
		path:      append([]string(nil), path...),
		command:   command,
		method:    method,
		humanPath: humanPath,
	}
	if contract, ok := contracts[key]; ok {
		leaf.contract = contract
		leaf.hasSpec = true
	}
	return leaf, true, ""
}

func hasPathPrefix(paths map[string]struct{}, prefix string) bool {
	for path := range paths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func runtimeLeafArgs(t *testing.T, leaf agentRuntimeLeaf) []string {
	t.Helper()

	args := append([]string(nil), leaf.path...)
	for _, param := range runtimeLeafPositionalParams(leaf) {
		args = append(args, runtimeValueForParameter(t, param))
	}

	var fields []openAPIInputField
	if leaf.hasSpec && leaf.contract.hasBody {
		fields = runtimeSelectedBodyFields(leaf.contract)
	}
	fields = runtimeMergeRequiredAnnotatedFields(leaf.command, fields)
	if cliCommandUsesRawBodyProxy(leaf.command) {
		for _, field := range fields {
			args = append(args, "-f", field.Name+"="+runtimeRawBodyValue(field))
		}
		return args
	}
	if len(fields) == 0 {
		return append(args, runtimeFallbackBodyFlagArgs(leaf.command)...)
	}

	for _, field := range fields {
		flagName := runtimeFlagNameForBodyField(leaf.command, field.Name)
		if flagName == "" {
			t.Fatalf("missing CLI flag for body field %q on %s", field.Name, strings.Join(leaf.path, " "))
		}
		args = append(args, runtimeFlagArgs(flagName, field)...)
	}
	return args
}

func runtimeMergeRequiredAnnotatedFields(command *cobra.Command, fields []openAPIInputField) []openAPIInputField {
	required := runtimeRequiredAnnotatedBodyFields(command)
	if len(required) == 0 {
		return fields
	}

	merged := append([]openAPIInputField(nil), fields...)
	present := make(map[string]struct{}, len(merged))
	for _, field := range merged {
		present[field.Name] = struct{}{}
	}
	for _, fieldName := range required {
		if _, ok := present[fieldName]; ok {
			continue
		}
		flagName := runtimeFlagNameForBodyField(command, fieldName)
		merged = append(merged, openAPIInputField{
			Name: fieldName,
			Kind: runtimeFlagKind(command.Flags().Lookup(flagName)),
		})
	}
	return merged
}

func runtimeFallbackBodyFlagArgs(command *cobra.Command) []string {
	if command == nil {
		return nil
	}

	fields := runtimeRequiredAnnotatedBodyFields(command)
	if len(fields) == 0 {
		fields = runtimeAnnotatedBodyFields(command)
	}
	args := make([]string, 0, len(fields)*2)
	for _, fieldName := range fields {
		flagName := runtimeFlagNameForBodyField(command, fieldName)
		if flagName == "" {
			continue
		}
		args = append(args, runtimeFlagArgs(flagName, openAPIInputField{
			Name: fieldName,
			Kind: runtimeFlagKind(command.Flags().Lookup(flagName)),
		})...)
	}
	return args
}

func runtimeLeafPositionalParams(leaf agentRuntimeLeaf) []string {
	if leaf.hasSpec && len(leaf.contract.spec.PositionalParams) > 0 {
		return append([]string(nil), leaf.contract.spec.PositionalParams...)
	}
	return bracketedUseParams(leaf.command.Use)
}

func runtimeSelectedBodyFields(contract openAPICommandContract) []openAPIInputField {
	if !contract.hasBody {
		return nil
	}

	if len(contract.requiredBody) > 0 {
		selected := make([]openAPIInputField, 0, len(contract.requiredBody))
		for _, name := range contract.requiredBody {
			for _, field := range contract.bodyFields {
				if field.Name == name {
					selected = append(selected, field)
					break
				}
			}
		}
		if len(selected) > 0 {
			return selected
		}
	}

	preferred := []string{
		"title",
		"name",
		"body",
		"content",
		"template",
		"description",
		"input_tokens",
		"repo_id",
		"workflow_ids",
		"event_types",
		"platform_access_allowed",
		"project_ai_platform_access_allowed",
	}
	for _, want := range preferred {
		for _, field := range contract.bodyFields {
			if field.Name == want {
				return []openAPIInputField{field}
			}
		}
	}
	if len(contract.bodyFields) == 0 {
		return nil
	}
	return []openAPIInputField{contract.bodyFields[0]}
}

func runtimeFlagNameForBodyField(command *cobra.Command, fieldName string) string {
	if command == nil {
		return ""
	}

	bestName := ""
	bestScore := -1
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		score := runtimeBodyFieldFlagScore(flag, fieldName)
		if score > bestScore {
			bestScore = score
			bestName = flag.Name
		}
	})
	if bestScore >= 0 {
		return bestName
	}

	if command.Flags().Lookup(fieldName) != nil {
		return fieldName
	}
	snake := snakeCaseParameterName(fieldName)
	if snake != "" && command.Flags().Lookup(snake) != nil {
		return snake
	}
	return ""
}

func runtimeRequiredAnnotatedBodyFields(command *cobra.Command) []string {
	if command == nil {
		return nil
	}

	required := make(map[string]struct{})
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag == nil || len(flag.Annotations) == 0 {
			return
		}
		if _, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]; !ok {
			return
		}
		for _, fieldName := range cliFlagBodyFields(flag) {
			required[fieldName] = struct{}{}
		}
	})
	return runtimePreferredAnnotatedBodyFields(required)
}

func runtimeAnnotatedBodyFields(command *cobra.Command) []string {
	if command == nil {
		return nil
	}

	all := make(map[string]struct{})
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		for _, fieldName := range cliFlagBodyFields(flag) {
			all[fieldName] = struct{}{}
		}
	})
	return runtimePreferredAnnotatedBodyFields(all)
}

func runtimePreferredAnnotatedBodyFields(fields map[string]struct{}) []string {
	if len(fields) == 0 {
		return nil
	}

	preferred := []string{
		"title",
		"name",
		"repository_url",
		"body",
		"description",
		"content",
		"template",
		"input_tokens",
	}
	selected := make([]string, 0, len(fields))
	for _, fieldName := range preferred {
		if _, ok := fields[fieldName]; ok {
			selected = append(selected, fieldName)
			delete(fields, fieldName)
		}
	}
	remaining := make([]string, 0, len(fields))
	for fieldName := range fields {
		remaining = append(remaining, fieldName)
	}
	slices.Sort(remaining)
	return append(selected, remaining...)
}

func runtimeBodyFieldFlagScore(flag *pflag.Flag, fieldName string) int {
	if flag == nil {
		return -1
	}
	for _, bodyField := range cliFlagBodyFields(flag) {
		if bodyField != fieldName {
			continue
		}
		score := 10
		if flag.Hidden {
			score--
		}
		if strings.Contains(flag.Name, "file") {
			score--
		}
		if flag.Name == fieldName {
			score += 2
		}
		if flag.Name == snakeCaseParameterName(fieldName) {
			score++
		}
		return score
	}
	return -1
}

func runtimeFlagKind(flag *pflag.Flag) flagValueKind {
	if flag == nil || flag.Value == nil {
		return flagValueString
	}
	switch strings.ToLower(strings.TrimSpace(flag.Value.Type())) {
	case "bool":
		return flagValueBool
	case "int64":
		return flagValueInt64
	case "float64":
		return flagValueFloat64
	case "stringslice", "stringarray":
		return flagValueStringSlice
	default:
		return flagValueString
	}
}

func runtimeFlagArgs(name string, field openAPIInputField) []string {
	switch field.Kind {
	case flagValueBool:
		return []string{"--" + name + "=true"}
	default:
		return []string{"--" + name, runtimeStringValueForField(field)}
	}
}

func runtimeRawBodyValue(field openAPIInputField) string {
	switch field.Kind {
	case flagValueBool:
		return "true"
	case flagValueInt64:
		return "1"
	case flagValueFloat64:
		return "1.5"
	case flagValueStringSlice:
		return `["` + runtimeStringValueForField(field) + `"]`
	default:
		return runtimeStringValueForField(field)
	}
}

func runtimeStringValueForField(field openAPIInputField) string {
	name := snakeCaseParameterName(field.Name)
	switch {
	case strings.HasSuffix(name, "_id"):
		return runtimeIDForName(name)
	case name == "slug":
		return "runtime-harness"
	case name == "priority":
		return "high"
	case name == "type":
		return "chore"
	case name == "status":
		return "Todo"
	case name == "url":
		return "https://example.com/runtime-harness"
	case name == "external_ref":
		return "ASE-178"
	case name == "external_id":
		return "runtime-harness"
	case name == "branch":
		return "main"
	case name == "schedule":
		return "0 * * * *"
	case strings.Contains(name, "cron"):
		return "0 * * * *"
	case name == "template":
		return "runtime harness template"
	case name == "content":
		return "runtime harness content"
	case name == "body":
		return "runtime harness body"
	case name == "description":
		return "runtime harness description"
	case name == "title":
		return "Runtime Harness"
	case name == "name":
		return "runtime-harness"
	case name == "input_tokens":
		return "1"
	case name == "workflow_ids":
		return runtimeHarnessWorkflowID
	case name == "event_types":
		return "ticket.created"
	case name == "platform_access_allowed":
		return "tickets.list"
	case name == "project_ai_platform_access_allowed":
		return "projects.update"
	default:
		switch field.Kind {
		case flagValueInt64:
			return "1"
		case flagValueFloat64:
			return "1.5"
		case flagValueStringSlice:
			return "runtime-harness"
		default:
			return "runtime-harness"
		}
	}
}

func runtimeValueForParameter(t *testing.T, name string) string {
	t.Helper()

	switch snakeCaseParameterName(name) {
	case "dir":
		root := t.TempDir()
		dir := filepath.Join(root, "runtime-skill")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("MkdirAll(%q) returned error: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("runtime harness skill\n"), 0o600); err != nil {
			t.Fatalf("WriteFile(skill bundle) returned error: %v", err)
		}
		return dir
	default:
		return runtimeIDForName(name)
	}
}

func runtimeIDForName(name string) string {
	switch snakeCaseParameterName(name) {
	case "organization_id", "org_id":
		return runtimeHarnessOrgID
	case "project_id":
		return runtimeHarnessProjectID
	case "ticket_id":
		return runtimeHarnessTicketID
	case "workflow_id":
		return runtimeHarnessWorkflowID
	case "agent_id":
		return runtimeHarnessAgentID
	case "run_id":
		return runtimeHarnessRunID
	case "repo_id":
		return runtimeHarnessRepoID
	case "scope_id":
		return runtimeHarnessScopeID
	case "rule_id":
		return runtimeHarnessRuleID
	case "status_id":
		return runtimeHarnessStatusID
	case "comment_id":
		return runtimeHarnessCommentID
	case "dependency_id":
		return runtimeHarnessDependencyID
	case "externallink_id", "external_link_id":
		return runtimeHarnessLinkID
	case "thread_id":
		return runtimeHarnessThreadID
	case "job_id":
		return runtimeHarnessJobID
	case "skill_id":
		return runtimeHarnessSkillID
	default:
		return "abababab-abab-abab-abab-abababababab"
	}
}

func runtimeExpectedRequestPath(leaf agentRuntimeLeaf, basePath string) string {
	path := strings.TrimPrefix(leaf.humanPath, "/api/v1")
	if strings.HasSuffix(basePath, "/platform") {
		path = "/api/v1/platform" + path
	} else {
		path = "/api/v1" + path
	}

	for _, placeholder := range bracketedPathParams(leaf.humanPath) {
		path = strings.ReplaceAll(path, "{"+placeholder+"}", runtimeIDForName(placeholder))
	}
	return path
}

func bracketedPathParams(path string) []string {
	items := make([]string, 0, 4)
	start := -1
	for index, r := range path {
		switch r {
		case '{':
			start = index + 1
		case '}':
			if start >= 0 && start < index {
				items = append(items, path[start:index])
			}
			start = -1
		}
	}
	return items
}
