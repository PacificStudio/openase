package workflow

import "testing"

func TestClassifyWorkflowUsesRoleSlugFirst(t *testing.T) {
	classification := ClassifyWorkflow(WorkflowClassificationInput{
		RoleSlug:     "product-manager",
		TypeLabel:    MustParseTypeLabel("Custom Label"),
		WorkflowName: "Anything",
	})

	if classification.Family != WorkflowFamilyPlanning {
		t.Fatalf("Family = %q, want %q", classification.Family, WorkflowFamilyPlanning)
	}
	if len(classification.Reasons) == 0 || classification.Reasons[0] != "matched explicit built-in role slug" {
		t.Fatalf("Reasons = %+v", classification.Reasons)
	}
}

func TestClassifyWorkflowUsesStatusSemantics(t *testing.T) {
	classification := ClassifyWorkflow(WorkflowClassificationInput{
		TypeLabel:         MustParseTypeLabel("Release Captain"),
		WorkflowName:      "Ship Lane",
		PickupStatusNames: []string{"Ready for Deploy"},
		FinishStatusNames: []string{"Released"},
	})

	if classification.Family != WorkflowFamilyDeploy {
		t.Fatalf("Family = %q, want %q", classification.Family, WorkflowFamilyDeploy)
	}
}

func TestClassifyWorkflowSupportsLegacyEnumLabels(t *testing.T) {
	classification := ClassifyWorkflow(WorkflowClassificationInput{
		TypeLabel: MustParseTypeLabel("refine-harness"),
	})

	if classification.Family != WorkflowFamilyHarness {
		t.Fatalf("Family = %q, want %q", classification.Family, WorkflowFamilyHarness)
	}
}

func TestClassifyWorkflowUsesNameHintsHarnessContentAndFallback(t *testing.T) {
	tests := []struct {
		name  string
		input WorkflowClassificationInput
		want  WorkflowFamily
	}{
		{
			name: "workflow name alias",
			input: WorkflowClassificationInput{
				WorkflowName: "report",
			},
			want: WorkflowFamilyReporting,
		},
		{
			name: "skill hint",
			input: WorkflowClassificationInput{
				SkillNames: []string{"security-scan"},
			},
			want: WorkflowFamilySecurity,
		},
		{
			name: "skill hint deploy",
			input: WorkflowClassificationInput{
				SkillNames: []string{"release"},
			},
			want: WorkflowFamilyDeploy,
		},
		{
			name: "harness content role slug",
			input: WorkflowClassificationInput{
				HarnessContent: "---\nworkflow:\n  role: qa-engineer\n---\n# QA\n",
			},
			want: WorkflowFamilyTest,
		},
		{
			name: "harness content keyword",
			input: WorkflowClassificationInput{
				HarnessContent: "This workflow focuses on rollout safety.",
			},
			want: WorkflowFamilyDeploy,
		},
		{
			name: "fallback unknown",
			input: WorkflowClassificationInput{
				WorkflowName: "Something bespoke",
			},
			want: WorkflowFamilyUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classification := ClassifyWorkflow(test.input)
			if classification.Family != test.want {
				t.Fatalf("Family = %q, want %q", classification.Family, test.want)
			}
			if len(classification.Reasons) == 0 {
				t.Fatal("expected classification reasons")
			}
		})
	}
}

func TestClassificationDefaultsReasonWhenEmpty(t *testing.T) {
	result := classification(WorkflowFamilyCoding, 0.5, "  ", "")
	if len(result.Reasons) != 1 || result.Reasons[0] != "no workflow family signal matched" {
		t.Fatalf("Reasons = %+v", result.Reasons)
	}
}

func TestClassifyByRoleSlugAndAliasRejectUnknownSignals(t *testing.T) {
	if _, _, ok := classifyByRoleSlug("unknown-role"); ok {
		t.Fatal("expected unknown role slug to be ignored")
	}
	if _, _, ok := classifyByAlias("", "type label"); ok {
		t.Fatal("expected empty alias input to be ignored")
	}
	if _, _, ok := classifyByAlias("unknownsignal", "type label"); ok {
		t.Fatal("expected unknown alias input to be ignored")
	}
}

func TestClassifyByStatusSemanticsFamilies(t *testing.T) {
	tests := []struct {
		name   string
		pickup []string
		finish []string
		want   WorkflowFamily
		ok     bool
	}{
		{name: "empty", want: "", ok: false},
		{name: "dispatcher", pickup: []string{"Backlog"}, finish: []string{"Backlog"}, want: WorkflowFamilyDispatcher, ok: true},
		{name: "review", pickup: []string{"PR Review"}, want: WorkflowFamilyReview, ok: true},
		{name: "test", pickup: []string{"QA Ready"}, want: WorkflowFamilyTest, ok: true},
		{name: "docs", pickup: []string{"Documentation"}, want: WorkflowFamilyDocs, ok: true},
		{name: "deploy", pickup: []string{"Ready for Deploy"}, want: WorkflowFamilyDeploy, ok: true},
		{name: "security", pickup: []string{"Security Audit"}, want: WorkflowFamilySecurity, ok: true},
		{name: "harness", pickup: []string{"Prompt Tuning"}, want: WorkflowFamilyHarness, ok: true},
		{name: "environment", pickup: []string{"环境修复"}, want: WorkflowFamilyEnvironment, ok: true},
		{name: "reporting", pickup: []string{"paper"}, want: WorkflowFamilyReporting, ok: true},
		{name: "research", pickup: []string{"Research lane"}, want: WorkflowFamilyResearch, ok: true},
		{name: "planning", pickup: []string{"需求分析"}, want: WorkflowFamilyPlanning, ok: true},
		{name: "coding", pickup: []string{"Backend implementation"}, want: WorkflowFamilyCoding, ok: true},
		{name: "unknown", pickup: []string{"Waiting"}, want: "", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _, ok := classifyByStatusSemantics(test.pickup, test.finish)
			if ok != test.ok {
				t.Fatalf("ok = %v, want %v", ok, test.ok)
			}
			if got != test.want {
				t.Fatalf("family = %q, want %q", got, test.want)
			}
		})
	}
}

func TestClassifyByHintsAndHarnessContentFallbacks(t *testing.T) {
	if _, _, ok := classifyByHints(nil, ""); ok {
		t.Fatal("expected empty hints to be ignored")
	}
	if _, _, ok := classifyByHints([]string{"mystery"}, "tmp/custom.txt"); ok {
		t.Fatal("expected unmatched hints to be ignored")
	}
	if _, _, ok := classifyByHarnessContent(""); ok {
		t.Fatal("expected empty harness content to be ignored")
	}
	if _, _, ok := classifyByHarnessContent("plain custom content"); ok {
		t.Fatal("expected unmatched harness content to be ignored")
	}
}

func TestExtractHarnessFrontmatterAndRoleSlugErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "missing frontmatter", content: "# No frontmatter", wantErr: true},
		{name: "empty frontmatter", content: "---\n\n---\n# Empty", wantErr: true},
		{name: "missing closing delimiter", content: "---\nworkflow:\n  role: qa-engineer", wantErr: true},
		{name: "valid crlf", content: "---\r\nworkflow:\r\n  role: qa-engineer\r\n---\r\n# OK", wantErr: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := extractHarnessFrontmatter(test.content)
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}

	if got := extractHarnessRoleSlug("---\nworkflow:\n  role: [\n---\n"); got != "" {
		t.Fatalf("role slug = %q, want empty", got)
	}
}

func TestContainsValueAndAnyAliasMatch(t *testing.T) {
	if containsValue([]string{"alpha", "beta"}, "gamma") {
		t.Fatal("expected containsValue to report false")
	}
	if anyAliasMatch([]string{"custom"}, WorkflowFamilyDeploy) {
		t.Fatal("expected anyAliasMatch to report false")
	}
}
