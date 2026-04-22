package projectpreset

import (
	"strings"
	"testing"
)

func TestParseYAMLHappyPath(t *testing.T) {
	preset, err := ParseYAML("testdata/fullstack.yaml", []byte(`
version: 1
preset:
  key: fullstack-default
  name: Fullstack Delivery Pipeline
statuses:
  - name: Todo
    stage: unstarted
  - name: Done
    stage: completed
workflows:
  - name: Fullstack Developer Workflow
    key: fullstack-developer
    type: coding
    role_slug: fullstack-developer
    pickup_statuses: [Todo]
    finish_statuses: [Done]
project_ai:
  skill_references:
    - skill: auto-harness
      files: [references/checklist.md]
`))
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}
	if preset.Meta.Key != "fullstack-default" || len(preset.Statuses) != 2 || len(preset.Workflows) != 1 {
		t.Fatalf("ParseYAML() = %+v", preset)
	}
	if !preset.Statuses[0].Default {
		t.Fatalf("expected first status to become default when none declared: %+v", preset.Statuses)
	}
	if preset.Workflows[0].MaxRetryAttempts != 3 || preset.Workflows[0].TimeoutMinutes != 60 || preset.Workflows[0].StallTimeoutMinutes != 5 {
		t.Fatalf("workflow defaults = %+v", preset.Workflows[0])
	}
	if len(preset.ProjectAI.SkillReferences) != 1 || preset.ProjectAI.SkillReferences[0].Skill != "auto-harness" {
		t.Fatalf("project ai refs = %+v", preset.ProjectAI.SkillReferences)
	}
}

func TestParseYAMLRejectsInvalidStage(t *testing.T) {
	_, err := ParseYAML("testdata/invalid-stage.yaml", []byte(`
version: 1
preset:
  key: invalid
  name: Invalid
statuses:
  - name: Todo
    stage: nope
workflows:
  - key: coding
    name: Coding Workflow
    type: coding
    pickup_statuses: [Todo]
    finish_statuses: [Todo]
`))
	if err == nil || !strings.Contains(err.Error(), "statuses[0].stage") {
		t.Fatalf("ParseYAML() invalid stage error = %v", err)
	}
}

func TestParseYAMLRejectsUnknownWorkflowStatusReference(t *testing.T) {
	_, err := ParseYAML("testdata/unknown-status.yaml", []byte(`
version: 1
preset:
  key: invalid
  name: Invalid
statuses:
  - name: Todo
    stage: unstarted
  - name: Done
    stage: completed
workflows:
  - key: coding
    name: Coding Workflow
    type: coding
    pickup_statuses: [Todo]
    finish_statuses: [Missing]
`))
	if err == nil || !strings.Contains(err.Error(), "references unknown status \"Missing\"") {
		t.Fatalf("ParseYAML() unknown status error = %v", err)
	}
}
