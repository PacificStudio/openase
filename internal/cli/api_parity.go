package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	cliAPICoverageAnnotationKey            = "openase.api_coverage"
	cliExpectedBodyFieldsAnnotationKey     = "openase.expected_body_fields"
	cliIgnoredBodyFieldsAnnotationKey      = "openase.ignored_body_fields"
	cliAllowedExtraBodyFieldsAnnotationKey = "openase.allowed_extra_body_fields"
	cliRawBodyProxyAnnotationKey           = "openase.raw_body_proxy"
	cliBodyFieldAnnotationKey              = "openase.body_field"
)

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

func markCLICommandExpectedBodyFields(command *cobra.Command, fields ...string) *cobra.Command {
	return markCLICommandFieldAnnotation(command, cliExpectedBodyFieldsAnnotationKey, fields...)
}

func markCLICommandIgnoredBodyFields(command *cobra.Command, fields ...string) *cobra.Command {
	return markCLICommandFieldAnnotation(command, cliIgnoredBodyFieldsAnnotationKey, fields...)
}

func markCLICommandAllowedExtraBodyFields(command *cobra.Command, fields ...string) *cobra.Command {
	return markCLICommandFieldAnnotation(command, cliAllowedExtraBodyFieldsAnnotationKey, fields...)
}

func markCLICommandRawBodyProxy(command *cobra.Command) *cobra.Command {
	if command == nil {
		return nil
	}
	if command.Annotations == nil {
		command.Annotations = make(map[string]string, 1)
	}
	command.Annotations[cliRawBodyProxyAnnotationKey] = "true"
	return command
}

func annotateCLICommandBodyFlag(command *cobra.Command, flagName string, bodyFields ...string) {
	if command == nil {
		return
	}
	flag := command.Flags().Lookup(flagName)
	if flag == nil {
		panic(fmt.Errorf("missing flag %q while annotating CLI body contract", flagName))
	}
	annotateCLIFlagBodyFields(flag, bodyFields...)
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

func cliCommandExpectedBodyFields(command *cobra.Command) map[string]struct{} {
	return cliCommandFieldSetAnnotation(command, cliExpectedBodyFieldsAnnotationKey)
}

func cliCommandIgnoredBodyFields(command *cobra.Command) map[string]struct{} {
	return cliCommandFieldSetAnnotation(command, cliIgnoredBodyFieldsAnnotationKey)
}

func cliCommandAllowedExtraBodyFields(command *cobra.Command) map[string]struct{} {
	return cliCommandFieldSetAnnotation(command, cliAllowedExtraBodyFieldsAnnotationKey)
}

func cliCommandUsesRawBodyProxy(command *cobra.Command) bool {
	if command == nil || len(command.Annotations) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(command.Annotations[cliRawBodyProxyAnnotationKey]), "true")
}

func cliFlagBodyFields(flag *pflag.Flag) []string {
	if flag == nil || len(flag.Annotations) == 0 {
		return nil
	}
	values := flag.Annotations[cliBodyFieldAnnotationKey]
	return append([]string(nil), values...)
}

func annotateCLIFlagBodyFields(flag *pflag.Flag, bodyFields ...string) {
	if flag == nil {
		return
	}
	normalized := normalizeCLIFieldNames(bodyFields...)
	if len(normalized) == 0 {
		return
	}
	if flag.Annotations == nil {
		flag.Annotations = make(map[string][]string, 1)
	}
	flag.Annotations[cliBodyFieldAnnotationKey] = appendNormalizedCLIFieldList(flag.Annotations[cliBodyFieldAnnotationKey], normalized...)
}

func markCLICommandFieldAnnotation(command *cobra.Command, key string, fields ...string) *cobra.Command {
	if command == nil {
		return nil
	}
	normalized := normalizeCLIFieldNames(fields...)
	if len(normalized) == 0 {
		return command
	}
	if command.Annotations == nil {
		command.Annotations = make(map[string]string, 1)
	}
	command.Annotations[key] = strings.Join(appendNormalizedCLIFieldList(parseCLIFieldAnnotation(command.Annotations[key]), normalized...), ",")
	return command
}

func cliCommandFieldSetAnnotation(command *cobra.Command, key string) map[string]struct{} {
	values := make(map[string]struct{})
	if command == nil || len(command.Annotations) == 0 {
		return values
	}
	for _, field := range parseCLIFieldAnnotation(command.Annotations[key]) {
		values[field] = struct{}{}
	}
	return values
}

func parseCLIFieldAnnotation(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeCLIFieldNames(strings.Split(value, ",")...)
}

func appendNormalizedCLIFieldList(existing []string, fields ...string) []string {
	merged := make(map[string]struct{}, len(existing)+len(fields))
	for _, item := range existing {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			merged[trimmed] = struct{}{}
		}
	}
	for _, item := range fields {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			merged[trimmed] = struct{}{}
		}
	}
	items := make([]string, 0, len(merged))
	for item := range merged {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func normalizeCLIFieldNames(fields ...string) []string {
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return appendNormalizedCLIFieldList(nil, normalized...)
}
