package projectpreset

import (
	"fmt"
	"strings"
	"unicode"

	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"go.yaml.in/yaml/v3"
)

type rawPresetDocument struct {
	Version   int                 `yaml:"version"`
	Preset    rawPresetMeta       `yaml:"preset"`
	Statuses  []rawPresetStatus   `yaml:"statuses"`
	Workflows []rawPresetWorkflow `yaml:"workflows"`
	ProjectAI rawPresetProjectAI  `yaml:"project_ai"`
}

type rawPresetMeta struct {
	Key         string `yaml:"key"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type rawPresetStatus struct {
	Name          string `yaml:"name"`
	Stage         string `yaml:"stage"`
	Color         string `yaml:"color"`
	Icon          string `yaml:"icon"`
	MaxActiveRuns *int   `yaml:"max_active_runs"`
	Default       bool   `yaml:"default"`
	Description   string `yaml:"description"`
}

type rawPresetWorkflow struct {
	Key                   string   `yaml:"key"`
	Name                  string   `yaml:"name"`
	Type                  string   `yaml:"type"`
	RoleSlug              string   `yaml:"role_slug"`
	RoleName              string   `yaml:"role_name"`
	RoleDescription       string   `yaml:"role_description"`
	PlatformAccessAllowed []string `yaml:"platform_access_allowed"`
	SkillNames            []string `yaml:"skill_names"`
	HarnessPath           string   `yaml:"harness_path"`
	HarnessContent        string   `yaml:"harness_content"`
	MaxConcurrent         *int     `yaml:"max_concurrent"`
	MaxRetryAttempts      *int     `yaml:"max_retry_attempts"`
	TimeoutMinutes        *int     `yaml:"timeout_minutes"`
	StallTimeoutMinutes   *int     `yaml:"stall_timeout_minutes"`
	PickupStatuses        []string `yaml:"pickup_statuses"`
	FinishStatuses        []string `yaml:"finish_statuses"`
}

type rawPresetProjectAI struct {
	SkillReferences []rawPresetSkillReference `yaml:"skill_references"`
}

type rawPresetSkillReference struct {
	Skill string   `yaml:"skill"`
	Files []string `yaml:"files"`
}

func ParseYAML(sourcePath string, content []byte) (Preset, error) {
	var raw rawPresetDocument
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Preset{}, fmt.Errorf("parse preset YAML %s: %w", sourcePath, err)
	}
	if raw.Version != 1 {
		return Preset{}, fmt.Errorf("parse preset YAML %s: unsupported version %d", sourcePath, raw.Version)
	}

	meta, err := parsePresetMeta(sourcePath, raw.Preset)
	if err != nil {
		return Preset{}, err
	}
	statuses, err := parseStatuses(sourcePath, raw.Statuses)
	if err != nil {
		return Preset{}, err
	}
	statusNames := make(map[string]struct{}, len(statuses))
	for _, item := range statuses {
		statusNames[strings.ToLower(item.Name)] = struct{}{}
	}
	workflows, err := parseWorkflows(sourcePath, raw.Workflows, statusNames)
	if err != nil {
		return Preset{}, err
	}
	projectAI, err := parseProjectAI(sourcePath, raw.ProjectAI)
	if err != nil {
		return Preset{}, err
	}

	return Preset{
		Version:   raw.Version,
		Meta:      meta,
		Statuses:  statuses,
		Workflows: workflows,
		ProjectAI: projectAI,
	}, nil
}

func parsePresetMeta(sourcePath string, raw rawPresetMeta) (PresetMeta, error) {
	key := strings.TrimSpace(raw.Key)
	if key == "" {
		return PresetMeta{}, fmt.Errorf("parse preset YAML %s: preset.key must not be empty", sourcePath)
	}
	if strings.ContainsFunc(key, unicode.IsSpace) {
		return PresetMeta{}, fmt.Errorf("parse preset YAML %s: preset.key must not contain whitespace", sourcePath)
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return PresetMeta{}, fmt.Errorf("parse preset YAML %s: preset.name must not be empty", sourcePath)
	}
	return PresetMeta{Key: key, Name: name, Description: strings.TrimSpace(raw.Description), SourcePath: sourcePath}, nil
}

func parseStatuses(sourcePath string, rawStatuses []rawPresetStatus) ([]Status, error) {
	if len(rawStatuses) == 0 {
		return nil, fmt.Errorf("parse preset YAML %s: statuses must not be empty", sourcePath)
	}
	explicitDefaultCount := 0
	for _, item := range rawStatuses {
		if item.Default {
			explicitDefaultCount++
		}
	}
	if explicitDefaultCount > 1 {
		return nil, fmt.Errorf("parse preset YAML %s: only one status may be marked default", sourcePath)
	}

	parsed := make([]Status, 0, len(rawStatuses))
	seenNames := make(map[string]struct{}, len(rawStatuses))
	for index, item := range rawStatuses {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, fmt.Errorf("parse preset YAML %s: statuses[%d].name must not be empty", sourcePath, index)
		}
		nameKey := strings.ToLower(name)
		if _, exists := seenNames[nameKey]; exists {
			return nil, fmt.Errorf("parse preset YAML %s: duplicate status name %q", sourcePath, name)
		}
		seenNames[nameKey] = struct{}{}
		stage, err := ticketingdomain.ParseStatusStage(strings.TrimSpace(item.Stage))
		if err != nil {
			return nil, fmt.Errorf("parse preset YAML %s: statuses[%d].stage: %w", sourcePath, index, err)
		}
		if item.MaxActiveRuns != nil && *item.MaxActiveRuns <= 0 {
			return nil, fmt.Errorf("parse preset YAML %s: statuses[%d].max_active_runs must be greater than 0", sourcePath, index)
		}
		isDefault := item.Default
		if explicitDefaultCount == 0 && index == 0 {
			isDefault = true
		}
		if isDefault && stage.IsTerminal() {
			return nil, fmt.Errorf("parse preset YAML %s: default status %q must use a non-terminal stage", sourcePath, name)
		}
		parsed = append(parsed, Status{
			Name:          name,
			Stage:         stage,
			Color:         normalizeColor(strings.TrimSpace(item.Color)),
			Icon:          strings.TrimSpace(item.Icon),
			MaxActiveRuns: item.MaxActiveRuns,
			Default:       isDefault,
			Description:   strings.TrimSpace(item.Description),
		})
	}
	return parsed, nil
}

func parseWorkflows(sourcePath string, rawWorkflows []rawPresetWorkflow, statusNames map[string]struct{}) ([]Workflow, error) {
	if len(rawWorkflows) == 0 {
		return nil, fmt.Errorf("parse preset YAML %s: workflows must not be empty", sourcePath)
	}
	parsed := make([]Workflow, 0, len(rawWorkflows))
	seenKeys := make(map[string]struct{}, len(rawWorkflows))
	seenNames := make(map[string]struct{}, len(rawWorkflows))
	for index, item := range rawWorkflows {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].name must not be empty", sourcePath, index)
		}
		key := firstNonEmpty(strings.TrimSpace(item.Key), strings.TrimSpace(item.RoleSlug), slugify(name))
		if key == "" {
			return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].key must not be empty", sourcePath, index)
		}
		keyKey := strings.ToLower(key)
		if _, exists := seenKeys[keyKey]; exists {
			return nil, fmt.Errorf("parse preset YAML %s: duplicate workflow key %q", sourcePath, key)
		}
		seenKeys[keyKey] = struct{}{}
		nameKey := strings.ToLower(name)
		if _, exists := seenNames[nameKey]; exists {
			return nil, fmt.Errorf("parse preset YAML %s: duplicate workflow name %q", sourcePath, name)
		}
		seenNames[nameKey] = struct{}{}
		workflowType, err := workflowdomain.ParseType(strings.TrimSpace(item.Type))
		if err != nil {
			return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].type: %w", sourcePath, index, err)
		}
		pickupStatusNames, err := parseStatusReferenceList(sourcePath, index, "pickup_statuses", item.PickupStatuses, statusNames)
		if err != nil {
			return nil, err
		}
		finishStatusNames, err := parseStatusReferenceList(sourcePath, index, "finish_statuses", item.FinishStatuses, statusNames)
		if err != nil {
			return nil, err
		}
		harnessPath, err := parseOptionalHarnessPath(sourcePath, index, item.HarnessPath)
		if err != nil {
			return nil, err
		}
		maxConcurrent, err := parseNonNegativeOptionalInt(sourcePath, index, "max_concurrent", item.MaxConcurrent, 0)
		if err != nil {
			return nil, err
		}
		maxRetryAttempts, err := parseNonNegativeOptionalInt(sourcePath, index, "max_retry_attempts", item.MaxRetryAttempts, 3)
		if err != nil {
			return nil, err
		}
		timeoutMinutes, err := parsePositiveOptionalInt(sourcePath, index, "timeout_minutes", item.TimeoutMinutes, 60)
		if err != nil {
			return nil, err
		}
		stallTimeoutMinutes, err := parsePositiveOptionalInt(sourcePath, index, "stall_timeout_minutes", item.StallTimeoutMinutes, 5)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, Workflow{
			Key:                   key,
			Name:                  name,
			Type:                  workflowType,
			RoleSlug:              strings.TrimSpace(item.RoleSlug),
			RoleName:              firstNonEmpty(strings.TrimSpace(item.RoleName), name),
			RoleDescription:       firstNonEmpty(strings.TrimSpace(item.RoleDescription), strings.TrimSpace(item.RoleName), name),
			PlatformAccessAllowed: normalizeStringList(item.PlatformAccessAllowed),
			SkillNames:            normalizeStringList(item.SkillNames),
			HarnessPath:           harnessPath,
			HarnessContent:        strings.TrimSpace(item.HarnessContent),
			MaxConcurrent:         maxConcurrent,
			MaxRetryAttempts:      maxRetryAttempts,
			TimeoutMinutes:        timeoutMinutes,
			StallTimeoutMinutes:   stallTimeoutMinutes,
			PickupStatusNames:     pickupStatusNames,
			FinishStatusNames:     finishStatusNames,
		})
	}
	return parsed, nil
}

func parseProjectAI(sourcePath string, raw rawPresetProjectAI) (ProjectAI, error) {
	refs := make([]SkillReference, 0, len(raw.SkillReferences))
	for index, item := range raw.SkillReferences {
		skill := strings.TrimSpace(item.Skill)
		if skill == "" {
			return ProjectAI{}, fmt.Errorf("parse preset YAML %s: project_ai.skill_references[%d].skill must not be empty", sourcePath, index)
		}
		files := normalizeStringList(item.Files)
		refs = append(refs, SkillReference{Skill: skill, Files: files})
	}
	return ProjectAI{SkillReferences: refs}, nil
}

func parseStatusReferenceList(sourcePath string, workflowIndex int, field string, raw []string, known map[string]struct{}) ([]string, error) {
	normalized := normalizeStringList(raw)
	if len(normalized) == 0 {
		return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].%s must not be empty", sourcePath, workflowIndex, field)
	}
	for _, name := range normalized {
		if _, ok := known[strings.ToLower(name)]; !ok {
			return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].%s references unknown status %q", sourcePath, workflowIndex, field, name)
		}
	}
	return normalized, nil
}

func parseOptionalHarnessPath(sourcePath string, workflowIndex int, raw string) (*string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.HasPrefix(trimmed, ".openase/harnesses/") {
		return nil, fmt.Errorf("parse preset YAML %s: workflows[%d].harness_path must stay under .openase/harnesses/", sourcePath, workflowIndex)
	}
	return &trimmed, nil
}

func parseNonNegativeOptionalInt(sourcePath string, workflowIndex int, field string, raw *int, fallback int) (int, error) {
	if raw == nil {
		return fallback, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("parse preset YAML %s: workflows[%d].%s must be greater than or equal to 0", sourcePath, workflowIndex, field)
	}
	return *raw, nil
}

func parsePositiveOptionalInt(sourcePath string, workflowIndex int, field string, raw *int, fallback int) (int, error) {
	if raw == nil {
		return fallback, nil
	}
	if *raw <= 0 {
		return 0, fmt.Errorf("parse preset YAML %s: workflows[%d].%s must be greater than 0", sourcePath, workflowIndex, field)
	}
	return *raw, nil
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normalizeColor(raw string) string {
	if raw == "" {
		return "#6B7280"
	}
	return raw
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func slugify(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastDash := false
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case lastDash:
			continue
		default:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
