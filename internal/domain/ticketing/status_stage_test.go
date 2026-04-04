package ticketing

import "testing"

func TestStatusStageBehavior(t *testing.T) {
	t.Run("string and validity", func(t *testing.T) {
		if got := StatusStageCompleted.String(); got != "completed" {
			t.Fatalf("StatusStageCompleted.String() = %q, want completed", got)
		}
		if !StatusStageStarted.IsValid() {
			t.Fatal("StatusStageStarted should be valid")
		}
		if StatusStage("unknown").IsValid() {
			t.Fatal("unknown stage should be invalid")
		}
		if !StatusStageCompleted.IsTerminal() {
			t.Fatal("completed stage should be terminal")
		}
		if StatusStageStarted.IsTerminal() {
			t.Fatal("started stage should not be terminal")
		}
	})

	t.Run("workflow stage permissions", func(t *testing.T) {
		if !StatusStageBacklog.AllowsWorkflowPickup() {
			t.Fatal("backlog should allow pickup")
		}
		if !StatusStageStarted.AllowsWorkflowPickup() {
			t.Fatal("started should allow pickup")
		}
		if StatusStageCompleted.AllowsWorkflowPickup() {
			t.Fatal("completed should not allow pickup")
		}
		if !StatusStageCanceled.AllowsWorkflowFinish() {
			t.Fatal("canceled should allow finish")
		}
		if StatusStageUnstarted.AllowsWorkflowFinish() {
			t.Fatal("unstarted should not allow finish")
		}
	})

	t.Run("parse stage", func(t *testing.T) {
		parsed, err := ParseStatusStage(" Completed ")
		if err != nil {
			t.Fatalf("ParseStatusStage returned error: %v", err)
		}
		if parsed != StatusStageCompleted {
			t.Fatalf("ParseStatusStage = %q, want %q", parsed, StatusStageCompleted)
		}
		if _, err := ParseStatusStage("done"); err == nil {
			t.Fatal("ParseStatusStage should reject legacy display names")
		}
	})

	t.Run("default template stage inference", func(t *testing.T) {
		testCases := map[string]StatusStage{
			"Backlog":     StatusStageBacklog,
			"Todo":        StatusStageUnstarted,
			"In Progress": StatusStageStarted,
			"in-review":   StatusStageStarted,
			"Done":        StatusStageCompleted,
			"Cancelled":   StatusStageCanceled,
		}
		for raw, want := range testCases {
			got, ok := DefaultTemplateStatusStage(raw)
			if !ok || got != want {
				t.Fatalf("DefaultTemplateStatusStage(%q) = %q, %t; want %q, true", raw, got, ok, want)
			}
		}
		if _, ok := DefaultTemplateStatusStage("unknown"); ok {
			t.Fatal("DefaultTemplateStatusStage should reject unknown names")
		}
	})

	t.Run("general stage inference", func(t *testing.T) {
		testCases := map[string]StatusStage{
			"backlog":        StatusStageBacklog,
			"ready for work": StatusStageUnstarted,
			"testing":        StatusStageStarted,
			"merged":         StatusStageCompleted,
			"archived":       StatusStageCanceled,
			"won't fix":      StatusStageCanceled,
		}
		for raw, want := range testCases {
			got, ok := InferStatusStageFromName(raw)
			if !ok || got != want {
				t.Fatalf("InferStatusStageFromName(%q) = %q, %t; want %q, true", raw, got, ok, want)
			}
		}
		if _, ok := InferStatusStageFromName("mystery"); ok {
			t.Fatal("InferStatusStageFromName should reject unknown names")
		}
	})
}
