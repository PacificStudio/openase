package catalog

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseProjectAIRetentionPolicyDefaultsAndRaw(t *testing.T) {
	t.Parallel()

	policy, err := ParseProjectAIRetentionPolicy(nil)
	if err != nil {
		t.Fatalf("ParseProjectAIRetentionPolicy(nil) error = %v", err)
	}
	if policy != (ProjectAIRetentionPolicy{}) {
		t.Fatalf("ParseProjectAIRetentionPolicy(nil) = %+v", policy)
	}

	policy, err = ParseProjectAIRetentionPolicy(&ProjectAIRetentionPolicyInput{
		Enabled:        testBoolPtr(true),
		KeepLatestN:    testIntPtr(5),
		KeepRecentDays: testIntPtr(14),
	})
	if err != nil {
		t.Fatalf("ParseProjectAIRetentionPolicy(success) error = %v", err)
	}
	if !policy.Enabled || policy.KeepLatestN != 5 || policy.KeepRecentDays != 14 {
		t.Fatalf("ParseProjectAIRetentionPolicy(success) = %+v", policy)
	}

	raw := policy.Raw()
	if raw.Enabled == nil || !*raw.Enabled {
		t.Fatalf("ProjectAIRetentionPolicy.Raw().Enabled = %+v", raw.Enabled)
	}
	if raw.KeepLatestN == nil || *raw.KeepLatestN != 5 {
		t.Fatalf("ProjectAIRetentionPolicy.Raw().KeepLatestN = %+v", raw.KeepLatestN)
	}
	if raw.KeepRecentDays == nil || *raw.KeepRecentDays != 14 {
		t.Fatalf("ProjectAIRetentionPolicy.Raw().KeepRecentDays = %+v", raw.KeepRecentDays)
	}

	zeroRaw := (ProjectAIRetentionPolicy{}).Raw()
	if zeroRaw.Enabled == nil || *zeroRaw.Enabled {
		t.Fatalf("zero ProjectAIRetentionPolicy.Raw().Enabled = %+v", zeroRaw.Enabled)
	}
	if zeroRaw.KeepLatestN == nil || *zeroRaw.KeepLatestN != 0 {
		t.Fatalf("zero ProjectAIRetentionPolicy.Raw().KeepLatestN = %+v", zeroRaw.KeepLatestN)
	}
	if zeroRaw.KeepRecentDays == nil || *zeroRaw.KeepRecentDays != 0 {
		t.Fatalf("zero ProjectAIRetentionPolicy.Raw().KeepRecentDays = %+v", zeroRaw.KeepRecentDays)
	}
}

func TestParseProjectAIRetentionPolicyValidation(t *testing.T) {
	t.Parallel()

	if _, err := ParseProjectAIRetentionPolicy(&ProjectAIRetentionPolicyInput{
		Enabled: testBoolPtr(true),
	}); err == nil {
		t.Fatal("ParseProjectAIRetentionPolicy(enabled without keep rules) expected validation error")
	}

	if _, err := ParseProjectAIRetentionPolicy(&ProjectAIRetentionPolicyInput{
		KeepLatestN: testIntPtr(-1),
	}); err == nil {
		t.Fatal("ParseProjectAIRetentionPolicy(negative keep_latest_n) expected validation error")
	}

	if _, err := ParseProjectAIRetentionPolicy(&ProjectAIRetentionPolicyInput{
		KeepRecentDays: testIntPtr(-1),
	}); err == nil {
		t.Fatal("ParseProjectAIRetentionPolicy(negative keep_recent_days) expected validation error")
	}
}

func TestParseCreateProjectProjectAIRetention(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	project, err := ParseCreateProject(orgID, ProjectInput{
		Name: "Retention Ready",
		Slug: "retention-ready",
		ProjectAIRetention: &ProjectAIRetentionPolicyInput{
			Enabled:     testBoolPtr(true),
			KeepLatestN: testIntPtr(3),
		},
	})
	if err != nil {
		t.Fatalf("ParseCreateProject(retention) error = %v", err)
	}
	if !project.ProjectAIRetention.Enabled || project.ProjectAIRetention.KeepLatestN != 3 || project.ProjectAIRetention.KeepRecentDays != 0 {
		t.Fatalf("ParseCreateProject(retention) = %+v", project.ProjectAIRetention)
	}
	if project.AgentRunSummaryPrompt != "" {
		t.Fatalf("ParseCreateProject(retention) prompt = %q, want empty string", project.AgentRunSummaryPrompt)
	}

	if _, err := ParseCreateProject(orgID, ProjectInput{
		Name: "Bad Retention",
		Slug: "bad-retention",
		ProjectAIRetention: &ProjectAIRetentionPolicyInput{
			Enabled: testBoolPtr(true),
		},
	}); err == nil {
		t.Fatal("ParseCreateProject(invalid retention) expected validation error")
	}
}

func TestParseUpdateProjectProjectAIRetention(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	orgID := uuid.New()

	project, err := ParseUpdateProject(projectID, orgID, ProjectInput{
		Name: "Retention Ready",
		Slug: "retention-ready",
		ProjectAIRetention: &ProjectAIRetentionPolicyInput{
			Enabled:        testBoolPtr(true),
			KeepLatestN:    testIntPtr(2),
			KeepRecentDays: testIntPtr(7),
		},
	})
	if err != nil {
		t.Fatalf("ParseUpdateProject(retention) error = %v", err)
	}
	if project.ID != projectID || project.OrganizationID != orgID {
		t.Fatalf("ParseUpdateProject(retention) ids = %+v", project)
	}
	if !project.ProjectAIRetention.Enabled || project.ProjectAIRetention.KeepLatestN != 2 || project.ProjectAIRetention.KeepRecentDays != 7 {
		t.Fatalf("ParseUpdateProject(retention) = %+v", project.ProjectAIRetention)
	}

	if _, err := ParseUpdateProject(projectID, orgID, ProjectInput{
		Name: "Bad Retention",
		Slug: "bad-retention",
		ProjectAIRetention: &ProjectAIRetentionPolicyInput{
			Enabled: testBoolPtr(true),
		},
	}); err == nil {
		t.Fatal("ParseUpdateProject(invalid retention) expected validation error")
	}
}

func testBoolPtr(value bool) *bool {
	return &value
}

func testIntPtr(value int) *int {
	return &value
}
