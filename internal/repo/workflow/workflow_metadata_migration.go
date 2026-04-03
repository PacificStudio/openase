package workflow

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

type legacyWorkflowHarnessMetadata struct {
	HasFrontmatter        bool
	Body                  string
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PlatformAccessAllowed []string
	SkillNames            []string
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || slices.Contains(cloned, trimmed) {
			continue
		}
		cloned = append(cloned, trimmed)
	}
	return cloned
}

func normalizeHarnessNewlines(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\r", "\n")
}

func parseWorkflowVersionStatusIDs(raw pgarray.StringArray) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		parsed, err := uuid.Parse(strings.TrimSpace(item))
		if err != nil {
			continue
		}
		ids = append(ids, parsed)
	}
	return ids
}

func formatWorkflowVersionStatusIDs(ids []uuid.UUID) pgarray.StringArray {
	items := make(pgarray.StringArray, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		items = append(items, id.String())
	}
	return items
}

func normalizePlatformAccessAllowed(raw []string) []string {
	supported := agentplatformdomain.SupportedAgentScopes()
	allowed := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || !slices.Contains(supported, trimmed) || slices.Contains(allowed, trimmed) {
			continue
		}
		allowed = append(allowed, trimmed)
	}
	return allowed
}

func defaultPlatformAccessAllowed(raw []string) []string {
	normalized := normalizePlatformAccessAllowed(raw)
	if len(normalized) > 0 {
		return normalized
	}
	return agentplatformdomain.DefaultAgentScopes()
}

func parseLegacyWorkflowHarnessMetadata(content string) (legacyWorkflowHarnessMetadata, error) {
	normalized := normalizeHarnessNewlines(content)
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return legacyWorkflowHarnessMetadata{
			Body:            normalized,
			RoleDescription: firstBodyParagraph(normalized),
		}, nil
	}

	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end == -1 {
		return legacyWorkflowHarnessMetadata{}, fmt.Errorf("legacy harness frontmatter is missing the closing --- delimiter")
	}

	var document struct {
		Workflow struct {
			Name string `yaml:"name"`
			Role string `yaml:"role"`
		} `yaml:"workflow"`
		PlatformAccess struct {
			Allowed []string `yaml:"allowed"`
		} `yaml:"platform_access"`
		Skills []string `yaml:"skills"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &document); err != nil {
		return legacyWorkflowHarnessMetadata{}, fmt.Errorf("parse legacy harness frontmatter: %w", err)
	}

	body := strings.Join(lines[end+1:], "\n")
	return legacyWorkflowHarnessMetadata{
		HasFrontmatter:        true,
		Body:                  body,
		RoleSlug:              strings.TrimSpace(document.Workflow.Role),
		RoleName:              strings.TrimSpace(document.Workflow.Name),
		RoleDescription:       firstBodyParagraph(body),
		PlatformAccessAllowed: normalizePlatformAccessAllowed(document.PlatformAccess.Allowed),
		SkillNames:            copyStrings(document.Skills),
	}, nil
}

func firstBodyParagraph(body string) string {
	paragraph := make([]string, 0)
	for _, line := range strings.Split(normalizeHarnessNewlines(body), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			if len(paragraph) > 0 {
				return strings.Join(paragraph, " ")
			}
		case strings.HasPrefix(trimmed, "#"):
			continue
		default:
			paragraph = append(paragraph, trimmed)
		}
	}
	return strings.Join(paragraph, " ")
}

func workflowNeedsMetadataMigration(item *ent.Workflow) bool {
	if item == nil {
		return false
	}
	return strings.TrimSpace(item.RoleName) == "" ||
		len(item.PlatformAccessAllowed) == 0
}

func workflowVersionNeedsMetadataMigration(item *ent.WorkflowVersion) bool {
	if item == nil {
		return false
	}
	normalized := normalizeHarnessNewlines(item.ContentMarkdown)
	return strings.HasPrefix(normalized, "---\n") ||
		strings.TrimSpace(item.Name) == "" ||
		strings.TrimSpace(item.HarnessPath) == "" ||
		len(item.PickupStatusIds) == 0 ||
		len(item.FinishStatusIds) == 0 ||
		len(item.PlatformAccessAllowed) == 0
}

func (r *EntRepository) ensureProjectWorkflowsMigrated(ctx context.Context, projectID uuid.UUID) error {
	ids, err := r.client.Workflow.Query().
		Where(entworkflow.ProjectIDEQ(projectID)).
		IDs(ctx)
	if err != nil {
		return fmt.Errorf("list workflows for metadata migration: %w", err)
	}
	for _, workflowID := range ids {
		if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
			return err
		}
	}
	return nil
}

func (r *EntRepository) ensureWorkflowMigrated(ctx context.Context, workflowID uuid.UUID) error {
	workflowItem, err := r.client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithVersions(func(query *ent.WorkflowVersionQuery) {
			query.Order(ent.Asc(entworkflowversion.FieldVersion))
		}).
		WithSkillBindings(func(query *ent.WorkflowSkillBindingQuery) {
			query.WithSkill()
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrWorkflowNotFound
		}
		return fmt.Errorf("load workflow for metadata migration: %w", err)
	}

	needsVersionMigration := false
	for _, versionItem := range workflowItem.Edges.Versions {
		if workflowVersionNeedsMetadataMigration(versionItem) {
			needsVersionMigration = true
			break
		}
	}
	if !workflowNeedsMetadataMigration(workflowItem) && !needsVersionMigration {
		return nil
	}

	currentContent := ""
	if len(workflowItem.Edges.Versions) > 0 {
		currentContent = workflowItem.Edges.Versions[len(workflowItem.Edges.Versions)-1].ContentMarkdown
	}
	legacyCurrent, err := parseLegacyWorkflowHarnessMetadata(currentContent)
	if err != nil {
		return fmt.Errorf("parse current workflow harness for metadata migration: %w", err)
	}

	nextWorkflow := mapWorkflow(workflowItem)
	if strings.TrimSpace(nextWorkflow.RoleSlug) == "" {
		nextWorkflow.RoleSlug = strings.TrimSpace(legacyCurrent.RoleSlug)
	}
	if strings.TrimSpace(nextWorkflow.RoleName) == "" {
		nextWorkflow.RoleName = strings.TrimSpace(legacyCurrent.RoleName)
	}
	if strings.TrimSpace(nextWorkflow.RoleName) == "" {
		nextWorkflow.RoleName = nextWorkflow.Name
	}
	if strings.TrimSpace(nextWorkflow.RoleSlug) == "" {
		nextWorkflow.RoleSlug = repoSlugify(nextWorkflow.RoleName)
	}
	if strings.TrimSpace(nextWorkflow.RoleDescription) == "" {
		nextWorkflow.RoleDescription = strings.TrimSpace(legacyCurrent.RoleDescription)
	}
	nextWorkflow.PlatformAccessAllowed = defaultPlatformAccessAllowed(firstNonEmptyStrings(
		copyStrings(nextWorkflow.PlatformAccessAllowed),
		legacyCurrent.PlatformAccessAllowed,
	))

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start workflow metadata migration tx: %w", err)
	}
	defer rollback(tx)

	if workflowNeedsMetadataMigration(workflowItem) {
		if _, err := tx.Workflow.UpdateOneID(workflowID).
			SetRoleSlug(strings.TrimSpace(nextWorkflow.RoleSlug)).
			SetRoleName(strings.TrimSpace(nextWorkflow.RoleName)).
			SetRoleDescription(strings.TrimSpace(nextWorkflow.RoleDescription)).
			SetPlatformAccessAllowed(pgarray.StringArray(copyStrings(nextWorkflow.PlatformAccessAllowed))).
			Save(ctx); err != nil {
			return fmt.Errorf("update workflow metadata: %w", err)
		}
	}

	if len(workflowItem.Edges.SkillBindings) == 0 && len(legacyCurrent.SkillNames) > 0 {
		if err := r.backfillWorkflowSkillBindings(ctx, tx, workflowItem.ProjectID, workflowID, legacyCurrent.SkillNames); err != nil {
			return err
		}
	}

	for _, versionItem := range workflowItem.Edges.Versions {
		legacyVersion, err := parseLegacyWorkflowHarnessMetadata(versionItem.ContentMarkdown)
		if err != nil {
			return fmt.Errorf("parse workflow version %d for metadata migration: %w", versionItem.Version, err)
		}
		if _, err := tx.WorkflowVersion.UpdateOneID(versionItem.ID).
			SetContentMarkdown(normalizeHarnessNewlines(legacyVersion.Body)).
			SetName(nextWorkflow.Name).
			SetType(nextWorkflow.Type.String()).
			SetRoleSlug(strings.TrimSpace(nextWorkflow.RoleSlug)).
			SetRoleName(strings.TrimSpace(nextWorkflow.RoleName)).
			SetRoleDescription(strings.TrimSpace(nextWorkflow.RoleDescription)).
			SetPickupStatusIds(formatWorkflowVersionStatusIDs(nextWorkflow.PickupStatusIDs)).
			SetFinishStatusIds(formatWorkflowVersionStatusIDs(nextWorkflow.FinishStatusIDs)).
			SetHarnessPath(nextWorkflow.HarnessPath).
			SetHooks(copyHooks(nextWorkflow.Hooks)).
			SetPlatformAccessAllowed(pgarray.StringArray(copyStrings(nextWorkflow.PlatformAccessAllowed))).
			SetMaxConcurrent(nextWorkflow.MaxConcurrent).
			SetMaxRetryAttempts(nextWorkflow.MaxRetryAttempts).
			SetTimeoutMinutes(nextWorkflow.TimeoutMinutes).
			SetStallTimeoutMinutes(nextWorkflow.StallTimeoutMinutes).
			SetIsActive(nextWorkflow.IsActive).
			SetContentHash(contentHash(normalizeHarnessNewlines(legacyVersion.Body))).
			Save(ctx); err != nil {
			return fmt.Errorf("update workflow version %d metadata: %w", versionItem.Version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workflow metadata migration tx: %w", err)
	}
	return nil
}

func (r *EntRepository) backfillWorkflowSkillBindings(
	ctx context.Context,
	tx *ent.Tx,
	projectID uuid.UUID,
	workflowID uuid.UUID,
	skillNames []string,
) error {
	names := copyStrings(skillNames)
	if len(names) == 0 {
		return nil
	}

	skills, err := tx.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.ArchivedAtIsNil(),
			entskill.NameIn(names...),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query skills for workflow metadata migration: %w", err)
	}
	for _, skillItem := range skills {
		if _, err := tx.WorkflowSkillBinding.Create().
			SetWorkflowID(workflowID).
			SetSkillID(skillItem.ID).
			Save(ctx); err != nil && !ent.IsConstraintError(err) {
			return fmt.Errorf("create migrated workflow skill binding %s: %w", skillItem.Name, err)
		}
	}
	return nil
}

func (r *EntRepository) createWorkflowVersionSnapshot(
	ctx context.Context,
	tx *ent.Tx,
	workflow domain.Workflow,
	version int,
	content string,
	createdBy string,
) (*ent.WorkflowVersion, error) {
	normalizedContent := normalizeHarnessNewlines(content)
	item, err := tx.WorkflowVersion.Create().
		SetWorkflowID(workflow.ID).
		SetVersion(version).
		SetContentMarkdown(normalizedContent).
		SetName(workflow.Name).
		SetType(workflow.Type.String()).
		SetRoleSlug(strings.TrimSpace(workflow.RoleSlug)).
		SetRoleName(strings.TrimSpace(workflowRoleName(workflow))).
		SetRoleDescription(strings.TrimSpace(workflow.RoleDescription)).
		SetPickupStatusIds(formatWorkflowVersionStatusIDs(workflow.PickupStatusIDs)).
		SetFinishStatusIds(formatWorkflowVersionStatusIDs(workflow.FinishStatusIDs)).
		SetHarnessPath(workflow.HarnessPath).
		SetHooks(copyHooks(workflow.Hooks)).
		SetPlatformAccessAllowed(pgarray.StringArray(copyStrings(workflow.PlatformAccessAllowed))).
		SetMaxConcurrent(workflow.MaxConcurrent).
		SetMaxRetryAttempts(workflow.MaxRetryAttempts).
		SetTimeoutMinutes(workflow.TimeoutMinutes).
		SetStallTimeoutMinutes(workflow.StallTimeoutMinutes).
		SetIsActive(workflow.IsActive).
		SetContentHash(contentHash(normalizedContent)).
		SetCreatedBy(resolveCreatedBy(createdBy)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create workflow version snapshot: %w", err)
	}
	return item, nil
}

func firstNonEmptyStrings(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func repoSlugify(raw string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if builder.Len() == 0 || lastDash {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
