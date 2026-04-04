package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

const cliAPICoverageAnnotationKey = "openase.api_coverage"

func markCLICommandAPICoverage(command *cobra.Command, method string, path string) *cobra.Command {
	if command == nil {
		return nil
	}
	if command.Annotations == nil {
		command.Annotations = make(map[string]string, 1)
	}
	command.Annotations[cliAPICoverageAnnotationKey] = contractKey(method, path)
	return command
}

func markCLICommandAPICoverageSpec(command *cobra.Command, spec openAPICommandSpec) *cobra.Command {
	return markCLICommandAPICoverage(command, spec.Method, spec.Path)
}

func cliCommandAPICoverageKey(command *cobra.Command) (string, bool) {
	if command == nil || len(command.Annotations) == 0 {
		return "", false
	}
	key := strings.TrimSpace(command.Annotations[cliAPICoverageAnnotationKey])
	if key == "" {
		return "", false
	}
	return key, true
}
