package projectpreset

import (
	"strings"
	"testing"

	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
)

func TestParsePresetMetaErrors(t *testing.T) {
	if _, err := parsePresetMeta("preset.yaml", rawPresetMeta{}); err == nil || !strings.Contains(err.Error(), "preset.key") {
		t.Fatalf("parsePresetMeta() empty key error = %v", err)
	}
	if _, err := parsePresetMeta("preset.yaml", rawPresetMeta{Key: "bad key", Name: "Preset"}); err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("parsePresetMeta() whitespace key error = %v", err)
	}
	if _, err := parsePresetMeta("preset.yaml", rawPresetMeta{Key: "good", Name: " "}); err == nil || !strings.Contains(err.Error(), "preset.name") {
		t.Fatalf("parsePresetMeta() empty name error = %v", err)
	}
}

func TestParseStatusesBranches(t *testing.T) {
	if _, err := parseStatuses("preset.yaml", nil); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("parseStatuses(nil) error = %v", err)
	}
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: " ", Stage: "unstarted"}}); err == nil || !strings.Contains(err.Error(), "statuses[0].name") {
		t.Fatalf("parseStatuses(empty name) error = %v", err)
	}
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Todo", Stage: "bad"}}); err == nil || !strings.Contains(err.Error(), "statuses[0].stage") {
		t.Fatalf("parseStatuses(invalid stage) error = %v", err)
	}
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Todo", Stage: "unstarted", Default: true}, {Name: "Backlog", Stage: "backlog", Default: true}}); err == nil || !strings.Contains(err.Error(), "only one status") {
		t.Fatalf("parseStatuses(multiple defaults) error = %v", err)
	}
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Todo", Stage: "unstarted"}, {Name: "todo", Stage: "started"}}); err == nil || !strings.Contains(err.Error(), "duplicate status name") {
		t.Fatalf("parseStatuses(duplicate) error = %v", err)
	}
	maxActiveRuns := 0
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Todo", Stage: "unstarted", MaxActiveRuns: &maxActiveRuns}}); err == nil || !strings.Contains(err.Error(), "max_active_runs") {
		t.Fatalf("parseStatuses(max_active_runs) error = %v", err)
	}
	if _, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Done", Stage: "completed", Default: true}}); err == nil || !strings.Contains(err.Error(), "non-terminal") {
		t.Fatalf("parseStatuses(default terminal) error = %v", err)
	}

	statuses, err := parseStatuses("preset.yaml", []rawPresetStatus{{Name: "Todo", Stage: "unstarted"}, {Name: "Done", Stage: "completed", Color: "#10B981"}})
	if err != nil {
		t.Fatalf("parseStatuses(valid) error = %v", err)
	}
	if statuses[0].Color != "#6B7280" || !statuses[0].Default || statuses[1].Color != "#10B981" {
		t.Fatalf("parseStatuses(valid) = %+v", statuses)
	}
}

func TestParseWorkflowsBranches(t *testing.T) {
	knownStatuses := map[string]struct{}{"todo": {}, "done": {}}
	if _, err := parseWorkflows("preset.yaml", nil, knownStatuses); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("parseWorkflows(nil) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Name: " ", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "workflows[0].name") {
		t.Fatalf("parseWorkflows(empty name) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "coding", Name: "A", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}, {Key: "coding", Name: "B", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "duplicate workflow key") {
		t.Fatalf("parseWorkflows(duplicate key) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}, {Key: "b", Name: "workflow", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "duplicate workflow name") {
		t.Fatalf("parseWorkflows(duplicate name) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: " ", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "workflows[0].type") {
		t.Fatalf("parseWorkflows(invalid type) error = %v", err)
	}
	badMax := -1
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", MaxConcurrent: &badMax, PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "max_concurrent") {
		t.Fatalf("parseWorkflows(max concurrent) error = %v", err)
	}
	badTimeout := 0
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", TimeoutMinutes: &badTimeout, PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "timeout_minutes") {
		t.Fatalf("parseWorkflows(timeout) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", HarnessPath: "workflow.md", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "harness_path") {
		t.Fatalf("parseWorkflows(harness path) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Name: "!!!", Type: "coding", PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "workflows[0].key") {
		t.Fatalf("parseWorkflows(empty derived key) error = %v", err)
	}
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", PickupStatuses: nil, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "pickup_statuses") {
		t.Fatalf("parseWorkflows(empty pickup) error = %v", err)
	}
	badRetryAttempts := -1
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", MaxRetryAttempts: &badRetryAttempts, PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "max_retry_attempts") {
		t.Fatalf("parseWorkflows(max retry attempts) error = %v", err)
	}
	badStallTimeout := 0
	if _, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{Key: "a", Name: "Workflow", Type: "coding", StallTimeoutMinutes: &badStallTimeout, PickupStatuses: []string{"Todo"}, FinishStatuses: []string{"Done"}}}, knownStatuses); err == nil || !strings.Contains(err.Error(), "stall_timeout_minutes") {
		t.Fatalf("parseWorkflows(stall timeout) error = %v", err)
	}

	maxConcurrent := 2
	maxRetryAttempts := 4
	timeoutMinutes := 45
	stallTimeoutMinutes := 7
	workflows, err := parseWorkflows("preset.yaml", []rawPresetWorkflow{{
		Name:                  "Feature Workflow",
		Type:                  "coding",
		RoleName:              "Backend Engineer",
		RoleDescription:       "Implements features",
		HarnessPath:           ".openase/harnesses/feature.md",
		HarnessContent:        "  # custom  ",
		PlatformAccessAllowed: []string{" tickets.list ", "tickets.list"},
		SkillNames:            []string{" commit ", "commit"},
		MaxConcurrent:         &maxConcurrent,
		MaxRetryAttempts:      &maxRetryAttempts,
		TimeoutMinutes:        &timeoutMinutes,
		StallTimeoutMinutes:   &stallTimeoutMinutes,
		PickupStatuses:        []string{"Todo"},
		FinishStatuses:        []string{"Done"},
	}}, knownStatuses)
	if err != nil {
		t.Fatalf("parseWorkflows(valid) error = %v", err)
	}
	if workflows[0].Key != "feature-workflow" || workflows[0].RoleName != "Backend Engineer" || workflows[0].MaxConcurrent != 2 || workflows[0].TimeoutMinutes != 45 || workflows[0].HarnessContent != "# custom" || len(workflows[0].PlatformAccessAllowed) != 1 || len(workflows[0].SkillNames) != 1 {
		t.Fatalf("parseWorkflows(valid) = %+v", workflows[0])
	}
	workflows, err = parseWorkflows("preset.yaml", []rawPresetWorkflow{{
		Name:           "Review Workflow",
		Type:           "coding",
		PickupStatuses: []string{"Todo"},
		FinishStatuses: []string{"Done"},
	}}, knownStatuses)
	if err != nil {
		t.Fatalf("parseWorkflows(fallback role name) error = %v", err)
	}
	if workflows[0].RoleName != "Review Workflow" || workflows[0].RoleDescription != "Review Workflow" {
		t.Fatalf("parseWorkflows(fallback role name) = %+v", workflows[0])
	}
}

func TestParseProjectAIAndHelpers(t *testing.T) {
	if _, err := parseProjectAI("preset.yaml", rawPresetProjectAI{SkillReferences: []rawPresetSkillReference{{Skill: " "}}}); err == nil || !strings.Contains(err.Error(), ".skill must not be empty") {
		t.Fatalf("parseProjectAI() error = %v", err)
	}
	projectAI, err := parseProjectAI("preset.yaml", rawPresetProjectAI{SkillReferences: []rawPresetSkillReference{{Skill: "auto-harness", Files: []string{" guide.md ", "guide.md"}}}})
	if err != nil {
		t.Fatalf("parseProjectAI(valid) error = %v", err)
	}
	if len(projectAI.SkillReferences[0].Files) != 1 {
		t.Fatalf("parseProjectAI(valid) = %+v", projectAI)
	}

	knownStatuses := map[string]struct{}{"todo": {}}
	if _, err := parseStatusReferenceList("preset.yaml", 0, "pickup_statuses", nil, knownStatuses); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("parseStatusReferenceList(empty) error = %v", err)
	}
	if _, err := parseStatusReferenceList("preset.yaml", 0, "pickup_statuses", []string{"Missing"}, knownStatuses); err == nil || !strings.Contains(err.Error(), "unknown status") {
		t.Fatalf("parseStatusReferenceList(missing) error = %v", err)
	}
	refs, err := parseStatusReferenceList("preset.yaml", 0, "pickup_statuses", []string{"Todo", "todo", " "}, knownStatuses)
	if err != nil || len(refs) != 1 || refs[0] != "Todo" {
		t.Fatalf("parseStatusReferenceList(valid) = %+v, %v", refs, err)
	}

	if path, err := parseOptionalHarnessPath("preset.yaml", 0, " "); err != nil || path != nil {
		t.Fatalf("parseOptionalHarnessPath(blank) = %v, %v", path, err)
	}
	if _, err := parseOptionalHarnessPath("preset.yaml", 0, "workflow.md"); err == nil || !strings.Contains(err.Error(), "harness_path") {
		t.Fatalf("parseOptionalHarnessPath(invalid) error = %v", err)
	}
	if path, err := parseOptionalHarnessPath("preset.yaml", 0, ".openase/harnesses/feature.md"); err != nil || path == nil || *path != ".openase/harnesses/feature.md" {
		t.Fatalf("parseOptionalHarnessPath(valid) = %v, %v", path, err)
	}

	if got, err := parseNonNegativeOptionalInt("preset.yaml", 0, "max_concurrent", nil, 3); err != nil || got != 3 {
		t.Fatalf("parseNonNegativeOptionalInt(nil) = %d, %v", got, err)
	}
	negative := -1
	if _, err := parseNonNegativeOptionalInt("preset.yaml", 0, "max_concurrent", &negative, 0); err == nil || !strings.Contains(err.Error(), "max_concurrent") {
		t.Fatalf("parseNonNegativeOptionalInt(negative) error = %v", err)
	}
	positive := 2
	if got, err := parseNonNegativeOptionalInt("preset.yaml", 0, "max_concurrent", &positive, 0); err != nil || got != 2 {
		t.Fatalf("parseNonNegativeOptionalInt(valid) = %d, %v", got, err)
	}

	if got, err := parsePositiveOptionalInt("preset.yaml", 0, "timeout_minutes", nil, 7); err != nil || got != 7 {
		t.Fatalf("parsePositiveOptionalInt(nil) = %d, %v", got, err)
	}
	zero := 0
	if _, err := parsePositiveOptionalInt("preset.yaml", 0, "timeout_minutes", &zero, 0); err == nil || !strings.Contains(err.Error(), "timeout_minutes") {
		t.Fatalf("parsePositiveOptionalInt(zero) error = %v", err)
	}
	validPositive := 9
	if got, err := parsePositiveOptionalInt("preset.yaml", 0, "timeout_minutes", &validPositive, 0); err != nil || got != 9 {
		t.Fatalf("parsePositiveOptionalInt(valid) = %d, %v", got, err)
	}

	if got := normalizeStringList([]string{"  Todo  ", "todo", "Done", ""}); len(got) != 2 || got[0] != "Todo" || got[1] != "Done" {
		t.Fatalf("normalizeStringList() = %+v", got)
	}
	if got := normalizeColor(""); got != "#6B7280" {
		t.Fatalf("normalizeColor(blank) = %q", got)
	}
	if got := normalizeColor("#123456"); got != "#123456" {
		t.Fatalf("normalizeColor(value) = %q", got)
	}
	if got := firstNonEmpty(" ", "todo", "done"); got != "todo" {
		t.Fatalf("firstNonEmpty() = %q", got)
	}
	if got := firstNonEmpty(" "); got != "" {
		t.Fatalf("firstNonEmpty(empty) = %q", got)
	}
	if got := slugify(" Feature Workflow / v2 "); got != "feature-workflow-v2" {
		t.Fatalf("slugify() = %q", got)
	}
	if got := slugify(" "); got != "" {
		t.Fatalf("slugify(blank) = %q", got)
	}

	if workflowdomain.Type(workflowsLabel("coding")) != workflowdomain.TypeLabel("coding") {
		t.Fatal("type alias guard failed")
	}
	if ticketingdomain.StatusStage(statusStageLabel("unstarted")) != ticketingdomain.StatusStageUnstarted {
		t.Fatal("status alias guard failed")
	}
}

func workflowsLabel(raw string) workflowdomain.TypeLabel {
	return workflowdomain.TypeLabel(raw)
}

func statusStageLabel(raw string) ticketingdomain.StatusStage {
	return ticketingdomain.StatusStage(raw)
}
