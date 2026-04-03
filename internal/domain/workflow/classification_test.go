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
