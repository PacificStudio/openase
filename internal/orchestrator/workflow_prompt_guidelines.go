package orchestrator

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed prompts/common-workflow-guidelines.md
var workflowPromptAssets embed.FS

var sharedWorkflowExecutionRules = mustReadWorkflowPromptAsset("prompts/common-workflow-guidelines.md")

func composeWorkflowDeveloperInstructions(renderedHarness string, platformContract string) string {
	sections := []string{
		strings.TrimSpace(renderedHarness),
		sharedWorkflowExecutionRules,
		strings.TrimSpace(platformContract),
	}
	nonEmpty := make([]string, 0, len(sections))
	for _, section := range sections {
		if section == "" {
			continue
		}
		nonEmpty = append(nonEmpty, section)
	}
	return strings.Join(nonEmpty, "\n\n")
}

func mustReadWorkflowPromptAsset(path string) string {
	data, err := workflowPromptAssets.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read workflow prompt asset %q: %v", path, err))
	}
	return strings.TrimSpace(string(data))
}
