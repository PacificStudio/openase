package cli

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestRootCLIBodyContracts(t *testing.T) {
	root := NewRootCommand("dev")
	contracts, err := loadOpenAPICommandContracts()
	if err != nil {
		t.Fatalf("load OpenAPI command contracts: %v", err)
	}

	failures := make([]string, 0)
	walkCLILeaves(root, []string{root.Name()}, func(path []string, command *cobra.Command) {
		if cliCommandUsesRawBodyProxy(command) {
			return
		}
		expected := expectedCLIBodyContractFields(command, contracts)
		if len(expected) == 0 {
			return
		}

		actual := annotatedCLICommandBodyFields(command)
		allowed := copyFieldSet(expected)
		for field := range cliCommandAllowedExtraBodyFields(command) {
			allowed[field] = struct{}{}
		}

		missing := diffFieldSets(expected, actual)
		extra := diffFieldSets(actual, allowed)
		if len(missing) == 0 && len(extra) == 0 {
			return
		}

		failures = append(failures, fmt.Sprintf(
			"%s\nmissing: %v\nextra: %v\nexpected: %v\nallowed: %v\nactual: %v",
			strings.Join(path, " "),
			missing,
			extra,
			sortedFieldSet(expected),
			sortedFieldSet(allowed),
			sortedFieldSet(actual),
		))
	})
	if len(failures) == 0 {
		return
	}
	sort.Strings(failures)
	t.Fatalf("CLI body contract drift detected.\n\n%s", strings.Join(failures, "\n\n"))
}

func expectedCLIBodyContractFields(command *cobra.Command, contracts map[string]openAPICommandContract) map[string]struct{} {
	expected := cliCommandExpectedBodyFields(command)
	if len(expected) == 0 {
		key, ok := cliCommandAPICoverageKey(command)
		if ok {
			if contract, ok := contracts[key]; ok {
				expected = toContractFieldSet(contract.bodyFields)
			}
		}
	}
	if len(expected) == 0 {
		return expected
	}
	for field := range cliCommandIgnoredBodyFields(command) {
		delete(expected, field)
	}
	return expected
}

func annotatedCLICommandBodyFields(command *cobra.Command) map[string]struct{} {
	fields := make(map[string]struct{})
	if command == nil {
		return fields
	}

	command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		for _, field := range cliFlagBodyFields(flag) {
			fields[field] = struct{}{}
		}
	})
	return fields
}

func toContractFieldSet(fields []openAPIInputField) map[string]struct{} {
	items := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name != "" {
			items[name] = struct{}{}
		}
	}
	return items
}

func diffFieldSets(expected map[string]struct{}, actual map[string]struct{}) []string {
	missing := make([]string, 0)
	for name := range expected {
		if _, ok := actual[name]; ok {
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return missing
}

func sortedFieldSet(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for name := range values {
		items = append(items, name)
	}
	sort.Strings(items)
	return items
}

func copyFieldSet(values map[string]struct{}) map[string]struct{} {
	clone := make(map[string]struct{}, len(values))
	for name := range values {
		clone[name] = struct{}{}
	}
	return clone
}
