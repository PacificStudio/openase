package workflow

import (
	"errors"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

type WorkflowFamily string

const (
	WorkflowFamilyPlanning    WorkflowFamily = "planning"
	WorkflowFamilyDispatcher  WorkflowFamily = "dispatcher"
	WorkflowFamilyCoding      WorkflowFamily = "coding"
	WorkflowFamilyReview      WorkflowFamily = "review"
	WorkflowFamilyTest        WorkflowFamily = "test"
	WorkflowFamilyDocs        WorkflowFamily = "docs"
	WorkflowFamilyDeploy      WorkflowFamily = "deploy"
	WorkflowFamilySecurity    WorkflowFamily = "security"
	WorkflowFamilyHarness     WorkflowFamily = "harness"
	WorkflowFamilyEnvironment WorkflowFamily = "environment"
	WorkflowFamilyResearch    WorkflowFamily = "research"
	WorkflowFamilyReporting   WorkflowFamily = "reporting"
	WorkflowFamilyUnknown     WorkflowFamily = "unknown"
)

type WorkflowClassification struct {
	Family     WorkflowFamily `json:"family"`
	Confidence float64        `json:"confidence"`
	Reasons    []string       `json:"reasons"`
}

type WorkflowClassificationInput struct {
	RoleSlug          string
	TypeLabel         TypeLabel
	WorkflowName      string
	PickupStatusNames []string
	FinishStatusNames []string
	SkillNames        []string
	HarnessPath       string
	HarnessContent    string
}

func ClassifyWorkflow(input WorkflowClassificationInput) WorkflowClassification {
	if family, reason, ok := classifyByRoleSlug(strings.TrimSpace(input.RoleSlug)); ok {
		return classification(family, 1.0, reason)
	}

	if family, reason, ok := classifyByAlias(input.TypeLabel.NormalizedKey(), "type label"); ok {
		return classification(family, 0.96, reason)
	}

	if family, reason, ok := classifyByAlias(normalizeSemanticKey(input.WorkflowName), "workflow name"); ok {
		return classification(family, 0.92, reason)
	}

	if family, reason, ok := classifyByStatusSemantics(input.PickupStatusNames, input.FinishStatusNames); ok {
		return classification(family, 0.84, reason)
	}

	if family, reasons, ok := classifyByHints(input.SkillNames, input.HarnessPath); ok {
		return classification(family, 0.76, reasons...)
	}

	if family, reasons, ok := classifyByHarnessContent(input.HarnessContent); ok {
		return classification(family, 0.68, reasons...)
	}

	return classification(WorkflowFamilyUnknown, 0.1, "no workflow family signal matched")
}

func classification(family WorkflowFamily, confidence float64, reasons ...string) WorkflowClassification {
	trimmed := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason != "" {
			trimmed = append(trimmed, reason)
		}
	}
	if len(trimmed) == 0 {
		trimmed = []string{"no workflow family signal matched"}
	}
	return WorkflowClassification{
		Family:     family,
		Confidence: confidence,
		Reasons:    trimmed,
	}
}

var roleSlugFamilies = map[string]WorkflowFamily{
	"productmanager":     WorkflowFamilyPlanning,
	"dispatcher":         WorkflowFamilyDispatcher,
	"fullstackdeveloper": WorkflowFamilyCoding,
	"frontendengineer":   WorkflowFamilyCoding,
	"backendengineer":    WorkflowFamilyCoding,
	"codereviewer":       WorkflowFamilyReview,
	"qaengineer":         WorkflowFamilyTest,
	"technicalwriter":    WorkflowFamilyDocs,
	"devopsengineer":     WorkflowFamilyDeploy,
	"securityengineer":   WorkflowFamilySecurity,
	"harnessoptimizer":   WorkflowFamilyHarness,
	"envprovisioner":     WorkflowFamilyEnvironment,
	"researchideation":   WorkflowFamilyResearch,
	"experimentrunner":   WorkflowFamilyResearch,
	"reportwriter":       WorkflowFamilyReporting,
	"marketanalyst":      WorkflowFamilyResearch,
	"dataanalyst":        WorkflowFamilyReporting,
}

var familyAliases = map[WorkflowFamily][]string{
	WorkflowFamilyPlanning: {
		"planning", "productmanager", "prd", "product", "plan", "requirements", "\u9700\u6c42\u5206\u6790", "\u89c4\u5212",
	},
	WorkflowFamilyDispatcher: {
		"dispatcher", "triage", "routing", "router", "dispatch", "\u8c03\u5ea6", "\u5206\u6d3e",
	},
	WorkflowFamilyCoding: {
		"coding", "coder", "developer", "implementation", "engineer", "backend", "frontend", "fullstack", "\u5f00\u53d1", "\u5b9e\u73b0",
	},
	WorkflowFamilyReview: {
		"review", "approval", "approver", "prreview", "reviewer", "\u5ba1\u67e5", "\u8bc4\u5ba1", "\u5ba1\u6838",
	},
	WorkflowFamilyTest: {
		"test", "testing", "qa", "verification", "verifier", "\u6d4b\u8bd5", "\u9a8c\u8bc1",
	},
	WorkflowFamilyDocs: {
		"doc", "docs", "documentation", "writer", "writeupdocs", "\u6587\u6863", "\u64b0\u5199",
	},
	WorkflowFamilyDeploy: {
		"deploy", "deployment", "release", "rollout", "ship", "\u4e0a\u7ebf", "\u90e8\u7f72", "\u53d1\u5e03",
	},
	WorkflowFamilySecurity: {
		"security", "audit", "scan", "secure", "\u5b89\u5168", "\u5ba1\u8ba1", "\u626b\u63cf",
	},
	WorkflowFamilyHarness: {
		"refineharness", "harness", "prompttuning", "workflowtune", "prompt", "\u4f18\u5316", "\u8c03\u4f18",
	},
	WorkflowFamilyEnvironment: {
		"environment", "env", "bootstrap", "provisioner", "machine", "setup", "repair", "\u73af\u5883", "\u914d\u7f6e", "\u4fee\u590d",
	},
	WorkflowFamilyResearch: {
		"research", "ideation", "investigate", "experiment", "trial", "study", "\u8c03\u7814", "\u7814\u7a76", "\u5b9e\u9a8c",
	},
	WorkflowFamilyReporting: {
		"report", "reporting", "writeup", "paper", "writer", "\u62a5\u544a", "\u8bba\u6587", "\u5199\u4f5c",
	},
}

func classifyByRoleSlug(raw string) (WorkflowFamily, string, bool) {
	if family, ok := roleSlugFamilies[normalizeSemanticKey(raw)]; ok {
		return family, "matched explicit built-in role slug", true
	}
	return "", "", false
}

func classifyByAlias(normalized string, signal string) (WorkflowFamily, string, bool) {
	if normalized == "" {
		return "", "", false
	}
	for family, aliases := range familyAliases {
		for _, alias := range aliases {
			if normalized == alias {
				return family, "matched " + signal + " alias", true
			}
		}
	}
	return "", "", false
}

func classifyByStatusSemantics(pickupStatuses []string, finishStatuses []string) (WorkflowFamily, string, bool) {
	pickupKeys := normalizedStatusKeys(pickupStatuses)
	finishKeys := normalizedStatusKeys(finishStatuses)
	if len(pickupKeys) == 0 && len(finishKeys) == 0 {
		return "", "", false
	}

	if containsValue(pickupKeys, normalizeSemanticKey("Backlog")) && containsValue(finishKeys, normalizeSemanticKey("Backlog")) {
		return WorkflowFamilyDispatcher, "matched backlog pickup/finish semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyReview) || anyAliasMatch(finishKeys, WorkflowFamilyReview) {
		return WorkflowFamilyReview, "matched review status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyTest) || anyAliasMatch(finishKeys, WorkflowFamilyTest) {
		return WorkflowFamilyTest, "matched test status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyDocs) || anyAliasMatch(finishKeys, WorkflowFamilyDocs) {
		return WorkflowFamilyDocs, "matched docs status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyDeploy) || anyAliasMatch(finishKeys, WorkflowFamilyDeploy) {
		return WorkflowFamilyDeploy, "matched deploy status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilySecurity) || anyAliasMatch(finishKeys, WorkflowFamilySecurity) {
		return WorkflowFamilySecurity, "matched security status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyHarness) || anyAliasMatch(finishKeys, WorkflowFamilyHarness) {
		return WorkflowFamilyHarness, "matched harness status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyEnvironment) || anyAliasMatch(finishKeys, WorkflowFamilyEnvironment) {
		return WorkflowFamilyEnvironment, "matched environment status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyReporting) || anyAliasMatch(finishKeys, WorkflowFamilyReporting) {
		return WorkflowFamilyReporting, "matched reporting status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyResearch) || anyAliasMatch(finishKeys, WorkflowFamilyResearch) {
		return WorkflowFamilyResearch, "matched research status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyPlanning) || anyAliasMatch(finishKeys, WorkflowFamilyPlanning) {
		return WorkflowFamilyPlanning, "matched planning status semantics", true
	}
	if anyAliasMatch(pickupKeys, WorkflowFamilyCoding) || anyAliasMatch(finishKeys, WorkflowFamilyCoding) {
		return WorkflowFamilyCoding, "matched coding status semantics", true
	}
	return "", "", false
}

func classifyByHints(skillNames []string, harnessPath string) (WorkflowFamily, []string, bool) {
	hints := make([]string, 0, len(skillNames)+1)
	for _, skillName := range skillNames {
		hints = append(hints, normalizeSemanticKey(skillName))
	}
	if trimmedPath := normalizeSemanticKey(harnessPath); trimmedPath != "" {
		hints = append(hints, trimmedPath)
	}
	if len(hints) == 0 {
		return "", nil, false
	}

	for family := range familyAliases {
		if !anyAliasMatch(hints, family) {
			continue
		}
		return family, []string{"matched harness path or bound skill hint"}, true
	}
	return "", nil, false
}

func classifyByHarnessContent(content string) (WorkflowFamily, []string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil, false
	}

	reasons := make([]string, 0, 2)
	if roleSlug := extractHarnessRoleSlug(trimmed); roleSlug != "" {
		if family, reason, ok := classifyByRoleSlug(roleSlug); ok {
			reasons = append(reasons, reason)
			return family, reasons, true
		}
	}

	contentKey := normalizeSemanticKey(trimmed)
	for family, aliases := range familyAliases {
		for _, alias := range aliases {
			if alias != "" && strings.Contains(contentKey, alias) {
				reasons = append(reasons, "matched harness content keyword hint")
				return family, reasons, true
			}
		}
	}
	return "", nil, false
}

func extractHarnessRoleSlug(content string) string {
	frontmatter, err := extractHarnessFrontmatter(content)
	if err != nil {
		return ""
	}

	var document struct {
		Workflow struct {
			Role string `yaml:"role"`
		} `yaml:"workflow"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return ""
	}
	return strings.TrimSpace(document.Workflow.Role)
}

func extractHarnessFrontmatter(content string) (string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("harness frontmatter is missing")
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "---" {
			continue
		}
		frontmatter := strings.Join(lines[1:index], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return "", errors.New("harness frontmatter is empty")
		}
		return frontmatter, nil
	}
	return "", errors.New("harness frontmatter closing delimiter not found")
}

func normalizedStatusKeys(items []string) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		key := normalizeSemanticKey(item)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func containsValue(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func anyAliasMatch(values []string, family WorkflowFamily) bool {
	aliases := familyAliases[family]
	for _, value := range values {
		for _, alias := range aliases {
			if strings.Contains(value, alias) || strings.Contains(alias, value) {
				return true
			}
		}
	}
	return false
}
