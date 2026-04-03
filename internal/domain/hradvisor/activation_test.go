package hradvisor

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseActivateRecommendation(t *testing.T) {
	projectID := uuid.New()
	createBootstrapTicket := true

	input, err := ParseActivateRecommendation(projectID, ActivateRecommendationRequest{
		RoleSlug:              " qa-engineer ",
		CreateBootstrapTicket: &createBootstrapTicket,
	})
	if err != nil {
		t.Fatalf("ParseActivateRecommendation() error = %v", err)
	}
	if input.ProjectID != projectID || input.RoleSlug != "qa-engineer" || !input.CreateBootstrapTicket {
		t.Fatalf("ParseActivateRecommendation() = %+v", input)
	}
}

func TestParseActivateRecommendationRejectsInvalidRoleSlug(t *testing.T) {
	_, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "QA Engineer"})
	if err == nil || !strings.Contains(err.Error(), "role_slug must be a lowercase slug") {
		t.Fatalf("ParseActivateRecommendation(invalid) error = %v", err)
	}
}

func TestParseActivateRecommendationRejectsEmptyRoleSlug(t *testing.T) {
	_, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "   "})
	if err == nil || !strings.Contains(err.Error(), "role_slug must not be empty") {
		t.Fatalf("ParseActivateRecommendation(empty) error = %v", err)
	}
}

func TestParseActivateRecommendationDefaultsBootstrapTicketToFalse(t *testing.T) {
	input, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "qa-engineer"})
	if err != nil {
		t.Fatalf("ParseActivateRecommendation() error = %v", err)
	}
	if input.CreateBootstrapTicket {
		t.Fatalf("expected bootstrap ticket default false, got %+v", input)
	}
}

func TestNormalizeActivationTemplate(t *testing.T) {
	template, err := NormalizeActivationTemplate(ActivationTemplate{
		RoleSlug:              " qa-engineer ",
		WorkflowName:          " QA Engineer ",
		WorkflowType:          " test ",
		HarnessPath:           " .openase/harnesses/roles/qa-engineer.md ",
		HarnessContent:        "\r\n# QA Engineer\r\n\r\nWrite automated regression coverage.\r\n",
		PickupStatusNames:     []string{" Todo ", "", "todo", "Todo"},
		FinishStatusNames:     []string{"Done", " ", " done "},
		PlatformAccessAllowed: []string{" tickets.list ", " ", "tickets.update.self", "tickets.list"},
		SkillNames:            []string{" openase-platform ", "", "write-test", "openase-platform"},
		Summary:               " Write automated regression coverage. ",
	})
	if err != nil {
		t.Fatalf("NormalizeActivationTemplate() error = %v", err)
	}
	if template.RoleSlug != "qa-engineer" || template.WorkflowName != "QA Engineer" || template.WorkflowType != "test" {
		t.Fatalf("NormalizeActivationTemplate() = %+v", template)
	}
	if template.HarnessPath != ".openase/harnesses/roles/qa-engineer.md" || template.HarnessContent != "# QA Engineer\n\nWrite automated regression coverage." {
		t.Fatalf("NormalizeActivationTemplate() normalized content = %+v", template)
	}
	if got := strings.Join(template.PickupStatusNames, ","); got != "Todo" {
		t.Fatalf("PickupStatusNames=%q", got)
	}
	if got := strings.Join(template.FinishStatusNames, ","); got != "Done" {
		t.Fatalf("FinishStatusNames=%q", got)
	}
	if got := strings.Join(template.PlatformAccessAllowed, ","); got != "tickets.list,tickets.update.self" {
		t.Fatalf("PlatformAccessAllowed=%q", got)
	}
	if got := strings.Join(template.SkillNames, ","); got != "openase-platform,write-test" {
		t.Fatalf("SkillNames=%q", got)
	}
	if template.Summary != "Write automated regression coverage." {
		t.Fatalf("Summary=%q", template.Summary)
	}
}

func TestNormalizeActivationTemplateRejectsMissingFields(t *testing.T) {
	cases := []struct {
		name     string
		template ActivationTemplate
		want     string
	}{
		{
			name: "missing role slug",
			template: ActivationTemplate{
				WorkflowName:      "QA Engineer",
				WorkflowType:      "test",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				HarnessContent:    "# QA Engineer",
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
			want: "role_slug must not be empty",
		},
		{
			name: "missing workflow name",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowType:      "test",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				HarnessContent:    "# QA Engineer",
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
			want: "workflow_name must not be empty",
		},
		{
			name: "missing workflow type",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowName:      "QA Engineer",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				HarnessContent:    "# QA Engineer",
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
			want: "workflow_type must not be empty",
		},
		{
			name: "missing harness path",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowName:      "QA Engineer",
				WorkflowType:      "test",
				HarnessContent:    "# QA Engineer",
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
			want: "harness_path must not be empty",
		},
		{
			name: "missing harness content",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowName:      "QA Engineer",
				WorkflowType:      "test",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				PickupStatusNames: []string{"Todo"},
				FinishStatusNames: []string{"Done"},
			},
			want: "harness_content must not be empty",
		},
		{
			name: "missing pickup statuses",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowName:      "QA Engineer",
				WorkflowType:      "test",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				HarnessContent:    "# QA Engineer",
				FinishStatusNames: []string{"Done"},
			},
			want: "pickup_status_names must not be empty",
		},
		{
			name: "missing finish statuses",
			template: ActivationTemplate{
				RoleSlug:          "qa-engineer",
				WorkflowName:      "QA Engineer",
				WorkflowType:      "test",
				HarnessPath:       ".openase/harnesses/roles/qa-engineer.md",
				HarnessContent:    "# QA Engineer",
				PickupStatusNames: []string{"Todo"},
			},
			want: "finish_status_names must not be empty",
		},
	}

	for _, testCase := range cases {
		if _, err := NormalizeActivationTemplate(testCase.template); err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Fatalf("%s error = %v, want substring %q", testCase.name, err, testCase.want)
		}
	}
}
