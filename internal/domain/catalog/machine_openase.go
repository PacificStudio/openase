package catalog

import (
	"path/filepath"
	"strings"
)

const machineOpenASEEnvKey = "OPENASE_REAL_BIN"

// ResolveMachineOpenASEBinaryPath returns the best-known runnable openase path
// for a machine. Prefer explicit machine env, then machine-channel telemetry,
// then the ssh-bootstrap default install location under the remote user's home.
func ResolveMachineOpenASEBinaryPath(machine Machine) *string {
	if value, ok := lookupMachineEnvironmentValue(machine.EnvVars, machineOpenASEEnvKey); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return &trimmed
		}
	}

	if telemetry := strings.TrimSpace(machineChannelOpenASEBinaryPath(machine.Resources)); telemetry != "" {
		return &telemetry
	}

	if machine.Host == LocalMachineHost || machine.SSHUser == nil {
		return nil
	}

	sshUser := strings.TrimSpace(*machine.SSHUser)
	if sshUser == "" {
		return nil
	}

	defaultPath := filepath.ToSlash(filepath.Join("/home", sshUser, ".openase", "bin", "openase"))
	return &defaultPath
}

func UpsertMachineEnvironmentValue(environment []string, key string, value string) []string {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return append([]string(nil), environment...)
	}

	trimmedValue := strings.TrimSpace(value)
	filtered := make([]string, 0, len(environment)+1)
	found := false
	for _, entry := range environment {
		name, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(strings.TrimSpace(name), trimmedKey) {
			if !found && trimmedValue != "" {
				filtered = append(filtered, trimmedKey+"="+trimmedValue)
				found = true
			}
			continue
		}
		filtered = append(filtered, entry)
	}
	if !found && trimmedValue != "" {
		filtered = append(filtered, trimmedKey+"="+trimmedValue)
	}
	return filtered
}

func lookupMachineEnvironmentValue(environment []string, key string) (string, bool) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", false
	}
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(strings.TrimSpace(name), trimmedKey) {
			return value, true
		}
	}
	return "", false
}

func machineChannelOpenASEBinaryPath(resources map[string]any) string {
	if len(resources) == 0 {
		return ""
	}
	raw, ok := resources["machine_channel"]
	if !ok {
		return ""
	}
	channel, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	value, _ := channel["openase_binary_path"].(string)
	return value
}
