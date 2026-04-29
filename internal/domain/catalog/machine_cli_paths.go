package catalog

import (
	"fmt"
	"sort"
	"strings"
)

type MachineAgentCLIPaths map[AgentProviderAdapterType]string

func parseMachineAgentCLIPaths(raw map[string]string) (MachineAgentCLIPaths, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	parsed := make(MachineAgentCLIPaths, len(raw))
	for key, value := range raw {
		adapterType, err := parseAgentProviderAdapterType(key)
		if err != nil {
			return nil, fmt.Errorf("agent_cli_paths[%q]: %w", key, err)
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return nil, fmt.Errorf("agent_cli_paths[%q] must not be empty", key)
		}
		parsed[adapterType] = trimmedValue
	}
	return parsed, nil
}

func cloneMachineAgentCLIPaths(input MachineAgentCLIPaths) MachineAgentCLIPaths {
	if len(input) == 0 {
		return nil
	}

	cloned := make(MachineAgentCLIPaths, len(input))
	for adapterType, path := range input {
		cloned[adapterType] = path
	}
	return cloned
}

func CloneMachineAgentCLIPaths(input MachineAgentCLIPaths) MachineAgentCLIPaths {
	return cloneMachineAgentCLIPaths(input)
}

func MachineAgentCLIPathsFromRaw(raw map[string]string) MachineAgentCLIPaths {
	if len(raw) == 0 {
		return nil
	}

	paths := make(MachineAgentCLIPaths, len(raw))
	for key, value := range raw {
		paths[AgentProviderAdapterType(key)] = value
	}
	return cloneMachineAgentCLIPaths(paths)
}

func (paths MachineAgentCLIPaths) Resolve(adapterType AgentProviderAdapterType) *string {
	if len(paths) == 0 {
		return nil
	}
	path := strings.TrimSpace(paths[adapterType])
	if path == "" {
		return nil
	}
	copied := path
	return &copied
}

func (paths MachineAgentCLIPaths) ToRawMap() map[string]string {
	if len(paths) == 0 {
		return nil
	}

	keys := make([]string, 0, len(paths))
	for adapterType := range paths {
		keys = append(keys, adapterType.String())
	}
	sort.Strings(keys)

	raw := make(map[string]string, len(keys))
	for _, key := range keys {
		path := strings.TrimSpace(paths[AgentProviderAdapterType(key)])
		if path == "" {
			continue
		}
		raw[key] = path
	}
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func ResolveMachineAgentCLIPath(machine Machine, adapterType AgentProviderAdapterType) *string {
	if resolved := machine.AgentCLIPaths.Resolve(adapterType); resolved != nil {
		return resolved
	}
	if len(machine.AgentCLIPaths) > 0 {
		return nil
	}
	if machine.AgentCLIPath == nil {
		return nil
	}
	path := strings.TrimSpace(*machine.AgentCLIPath)
	if path == "" {
		return nil
	}
	return &path
}

func ResolveProviderMachineAgentCLIPath(item AgentProvider) *string {
	if resolved := item.MachineAgentCLIPaths.Resolve(item.AdapterType); resolved != nil {
		return resolved
	}
	if len(item.MachineAgentCLIPaths) > 0 {
		return nil
	}
	if item.MachineAgentCLIPath == nil {
		return nil
	}
	path := strings.TrimSpace(*item.MachineAgentCLIPath)
	if path == "" {
		return nil
	}
	return &path
}
