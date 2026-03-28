package catalog

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
)

type MachineReachability struct {
	CheckedAt    time.Time
	Transport    string
	Reachable    bool
	LatencyMS    int64
	FailureCause string
}

type MachineSystemResources struct {
	CollectedAt            time.Time
	CPUCores               int
	CPUUsagePercent        float64
	MemoryTotalGB          float64
	MemoryUsedGB           float64
	MemoryAvailableGB      float64
	MemoryAvailablePercent float64
	DiskTotalGB            float64
	DiskAvailableGB        float64
	DiskAvailablePercent   float64
}

type MachineGPU struct {
	Index              int
	Name               string
	MemoryTotalGB      float64
	MemoryUsedGB       float64
	UtilizationPercent float64
}

type MachineGPUResources struct {
	CollectedAt time.Time
	Available   bool
	GPUs        []MachineGPU
}

type MachineAgentAuthStatus string

const (
	MachineAgentAuthStatusUnknown     MachineAgentAuthStatus = "unknown"
	MachineAgentAuthStatusLoggedIn    MachineAgentAuthStatus = "logged_in"
	MachineAgentAuthStatusNotLoggedIn MachineAgentAuthStatus = "not_logged_in"
)

type MachineAgentAuthMode string

const (
	MachineAgentAuthModeUnknown MachineAgentAuthMode = "unknown"
	MachineAgentAuthModeLogin   MachineAgentAuthMode = "login"
	MachineAgentAuthModeAPIKey  MachineAgentAuthMode = "api_key"
)

type MachineAgentCLI struct {
	Name       string
	Installed  bool
	Version    string
	AuthStatus MachineAgentAuthStatus
	AuthMode   MachineAgentAuthMode
	Ready      bool
}

type MachineAgentEnvironment struct {
	CollectedAt  time.Time
	Dispatchable bool
	CLIs         []MachineAgentCLI
}

type MachineGitAudit struct {
	Installed bool
	UserName  string
	UserEmail string
}

type MachineGitHubCLIAudit struct {
	Installed  bool
	AuthStatus MachineAgentAuthStatus
}

type MachineNetworkAudit struct {
	GitHubReachable bool
	PyPIReachable   bool
	NPMReachable    bool
}

type MachineFullAudit struct {
	CollectedAt      time.Time
	Git              MachineGitAudit
	GitHubCLI        MachineGitHubCLIAudit
	GitHubTokenProbe githubauthdomain.TokenProbe
	Network          MachineNetworkAudit
}

func ParseMachineSystemResources(raw string, collectedAt time.Time) (MachineSystemResources, error) {
	values, err := parseMachineMetricLines(raw)
	if err != nil {
		return MachineSystemResources{}, err
	}

	cpuCores, err := parseMetricInt(values, "cpu_cores")
	if err != nil {
		return MachineSystemResources{}, err
	}
	cpuUsagePercent, err := parseMetricFloat(values, "cpu_usage_percent")
	if err != nil {
		return MachineSystemResources{}, err
	}
	memTotalKB, err := parseMetricFloat(values, "memory_total_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	memAvailableKB, err := parseMetricFloat(values, "memory_available_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	diskTotalKB, err := parseMetricFloat(values, "disk_total_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}
	diskAvailableKB, err := parseMetricFloat(values, "disk_available_kb")
	if err != nil {
		return MachineSystemResources{}, err
	}

	memoryTotalGB := kilobytesToGigabytes(memTotalKB)
	memoryAvailableGB := kilobytesToGigabytes(memAvailableKB)
	memoryUsedGB := roundTwoDecimals(memoryTotalGB - memoryAvailableGB)
	diskTotalGB := kilobytesToGigabytes(diskTotalKB)
	diskAvailableGB := kilobytesToGigabytes(diskAvailableKB)

	return MachineSystemResources{
		CollectedAt:            collectedAt.UTC(),
		CPUCores:               cpuCores,
		CPUUsagePercent:        roundTwoDecimals(cpuUsagePercent),
		MemoryTotalGB:          memoryTotalGB,
		MemoryUsedGB:           memoryUsedGB,
		MemoryAvailableGB:      memoryAvailableGB,
		MemoryAvailablePercent: percentage(memoryAvailableGB, memoryTotalGB),
		DiskTotalGB:            diskTotalGB,
		DiskAvailableGB:        diskAvailableGB,
		DiskAvailablePercent:   percentage(diskAvailableGB, diskTotalGB),
	}, nil
}

func ParseMachineGPUResources(raw string, collectedAt time.Time) (MachineGPUResources, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "no_gpu") {
		return MachineGPUResources{
			CollectedAt: collectedAt.UTC(),
			Available:   false,
			GPUs:        nil,
		}, nil
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return MachineGPUResources{}, fmt.Errorf("parse gpu metrics csv: %w", err)
	}

	gpus := make([]MachineGPU, 0, len(records))
	for index, record := range records {
		if len(record) != 5 {
			return MachineGPUResources{}, fmt.Errorf("gpu metrics row %d must have 5 columns", index)
		}
		gpuIndex, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu index on row %d: %w", index, err)
		}
		memoryTotalMB, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu memory total on row %d: %w", index, err)
		}
		memoryUsedMB, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu memory used on row %d: %w", index, err)
		}
		utilizationPercent, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		if err != nil {
			return MachineGPUResources{}, fmt.Errorf("parse gpu utilization on row %d: %w", index, err)
		}

		gpus = append(gpus, MachineGPU{
			Index:              gpuIndex,
			Name:               strings.TrimSpace(record[1]),
			MemoryTotalGB:      roundTwoDecimals(memoryTotalMB / 1024.0),
			MemoryUsedGB:       roundTwoDecimals(memoryUsedMB / 1024.0),
			UtilizationPercent: roundTwoDecimals(utilizationPercent),
		})
	}

	return MachineGPUResources{
		CollectedAt: collectedAt.UTC(),
		Available:   true,
		GPUs:        gpus,
	}, nil
}

func ParseMachineAgentEnvironment(raw string, collectedAt time.Time) (MachineAgentEnvironment, error) {
	records, err := parseMachineTabularRecords(raw)
	if err != nil {
		return MachineAgentEnvironment{}, err
	}

	parsed := make(map[string]MachineAgentCLI, len(records))
	for index, record := range records {
		if len(record) != 4 && len(record) != 5 {
			return MachineAgentEnvironment{}, fmt.Errorf("agent environment row %d must have 4 or 5 columns", index)
		}

		name := strings.TrimSpace(record[0])
		if name == "" {
			return MachineAgentEnvironment{}, fmt.Errorf("agent environment row %d is missing cli name", index)
		}
		if _, exists := parsed[name]; exists {
			return MachineAgentEnvironment{}, fmt.Errorf("agent environment row %d duplicates cli %q", index, name)
		}

		installed, err := strconv.ParseBool(strings.TrimSpace(record[1]))
		if err != nil {
			return MachineAgentEnvironment{}, fmt.Errorf("parse agent environment installed on row %d: %w", index, err)
		}
		authStatus, err := parseMachineAgentAuthStatus(record[3])
		if err != nil {
			return MachineAgentEnvironment{}, fmt.Errorf("parse agent environment auth status on row %d: %w", index, err)
		}
		authMode := MachineAgentAuthModeUnknown
		if len(record) == 5 {
			authMode, err = parseMachineAgentAuthMode(record[4])
			if err != nil {
				return MachineAgentEnvironment{}, fmt.Errorf("parse agent environment auth mode on row %d: %w", index, err)
			}
		}

		parsed[name] = MachineAgentCLI{
			Name:       name,
			Installed:  installed,
			Version:    strings.TrimSpace(record[2]),
			AuthStatus: authStatus,
			AuthMode:   authMode,
			Ready:      installed && (authMode == MachineAgentAuthModeAPIKey || authStatus != MachineAgentAuthStatusNotLoggedIn),
		}
	}

	clis := make([]MachineAgentCLI, 0, 3)
	dispatchable := false
	for _, name := range []string{"claude_code", "codex", "gemini"} {
		cli, ok := parsed[name]
		if !ok {
			return MachineAgentEnvironment{}, fmt.Errorf("missing agent environment entry %q", name)
		}
		clis = append(clis, cli)
		if cli.Ready {
			dispatchable = true
		}
	}

	return MachineAgentEnvironment{
		CollectedAt:  collectedAt.UTC(),
		Dispatchable: dispatchable,
		CLIs:         clis,
	}, nil
}

func ParseMachineFullAudit(raw string, collectedAt time.Time) (MachineFullAudit, error) {
	records, err := parseMachineTabularRecords(raw)
	if err != nil {
		return MachineFullAudit{}, err
	}

	var (
		gitFound         bool
		ghFound          bool
		githubProbeFound bool
		networkFound     bool
		audit            = MachineFullAudit{CollectedAt: collectedAt.UTC(), GitHubTokenProbe: githubauthdomain.MissingProbe()}
	)

	for index, record := range records {
		switch strings.TrimSpace(record[0]) {
		case "git":
			if len(record) != 4 {
				return MachineFullAudit{}, fmt.Errorf("git audit row %d must have 4 columns", index)
			}
			installed, err := strconv.ParseBool(strings.TrimSpace(record[1]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse git audit installed on row %d: %w", index, err)
			}
			audit.Git = MachineGitAudit{
				Installed: installed,
				UserName:  strings.TrimSpace(record[2]),
				UserEmail: strings.TrimSpace(record[3]),
			}
			gitFound = true
		case "gh_cli":
			if len(record) != 3 {
				return MachineFullAudit{}, fmt.Errorf("gh_cli audit row %d must have 3 columns", index)
			}
			installed, err := strconv.ParseBool(strings.TrimSpace(record[1]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse gh_cli audit installed on row %d: %w", index, err)
			}
			authStatus, err := parseMachineAgentAuthStatus(record[2])
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse gh_cli audit auth status on row %d: %w", index, err)
			}
			audit.GitHubCLI = MachineGitHubCLIAudit{
				Installed:  installed,
				AuthStatus: authStatus,
			}
			ghFound = true
		case "network":
			if len(record) != 4 {
				return MachineFullAudit{}, fmt.Errorf("network audit row %d must have 4 columns", index)
			}
			githubReachable, err := strconv.ParseBool(strings.TrimSpace(record[1]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse network github reachability on row %d: %w", index, err)
			}
			pypiReachable, err := strconv.ParseBool(strings.TrimSpace(record[2]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse network pypi reachability on row %d: %w", index, err)
			}
			npmReachable, err := strconv.ParseBool(strings.TrimSpace(record[3]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse network npm reachability on row %d: %w", index, err)
			}
			audit.Network = MachineNetworkAudit{
				GitHubReachable: githubReachable,
				PyPIReachable:   pypiReachable,
				NPMReachable:    npmReachable,
			}
			networkFound = true
		case "github_token_probe":
			if len(record) != 7 {
				return MachineFullAudit{}, fmt.Errorf("github_token_probe row %d must have 7 columns", index)
			}
			configured, err := strconv.ParseBool(strings.TrimSpace(record[1]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse github_token_probe configured on row %d: %w", index, err)
			}
			valid, err := strconv.ParseBool(strings.TrimSpace(record[3]))
			if err != nil {
				return MachineFullAudit{}, fmt.Errorf("parse github_token_probe valid on row %d: %w", index, err)
			}
			state := githubauthdomain.ProbeState(strings.TrimSpace(record[2]))
			if !state.IsValid() {
				return MachineFullAudit{}, fmt.Errorf("parse github_token_probe state on row %d: invalid state %q", index, strings.TrimSpace(record[2]))
			}
			repoAccess := githubauthdomain.RepoAccess(strings.TrimSpace(record[5]))
			if !repoAccess.IsValid() {
				return MachineFullAudit{}, fmt.Errorf("parse github_token_probe repo access on row %d: invalid repo access %q", index, strings.TrimSpace(record[5]))
			}
			permissions := parseDelimitedMachinePermissions(record[4])
			checkedAt := collectedAt.UTC()
			audit.GitHubTokenProbe = githubauthdomain.TokenProbe{
				State:       state,
				Configured:  configured,
				Valid:       valid,
				Permissions: permissions,
				RepoAccess:  repoAccess,
				CheckedAt:   &checkedAt,
				LastError:   strings.TrimSpace(record[6]),
			}
			githubProbeFound = true
		default:
			return MachineFullAudit{}, fmt.Errorf("unknown machine audit row %q", strings.TrimSpace(record[0]))
		}
	}

	if !gitFound {
		return MachineFullAudit{}, fmt.Errorf("missing machine full audit entry %q", "git")
	}
	if !ghFound {
		return MachineFullAudit{}, fmt.Errorf("missing machine full audit entry %q", "gh_cli")
	}
	if !networkFound {
		return MachineFullAudit{}, fmt.Errorf("missing machine full audit entry %q", "network")
	}
	if !githubProbeFound {
		audit.GitHubTokenProbe = githubauthdomain.MissingProbe()
		checkedAt := collectedAt.UTC()
		audit.GitHubTokenProbe.CheckedAt = &checkedAt
	}

	return audit, nil
}

func parseDelimitedMachinePermissions(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	permissions := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		permissions = append(permissions, item)
	}
	return permissions
}

func parseMachineMetricLines(raw string) (map[string]string, error) {
	values := map[string]string{}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("machine metric line %q must be KEY=VALUE", trimmed)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return values, nil
}

func parseMachineTabularRecords(raw string) ([][]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("machine tabular payload must not be empty")
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.Comma = '\t'
	reader.TrimLeadingSpace = false
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse machine tabular payload: %w", err)
	}
	return records, nil
}

func parseMetricInt(values map[string]string, key string) (int, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing machine metric %q", key)
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse machine metric %q: %w", key, err)
	}
	return parsed, nil
}

func parseMetricFloat(values map[string]string, key string) (float64, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing machine metric %q", key)
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse machine metric %q: %w", key, err)
	}
	return parsed, nil
}

func parseMachineAgentAuthStatus(raw string) (MachineAgentAuthStatus, error) {
	status := MachineAgentAuthStatus(strings.ToLower(strings.TrimSpace(raw)))
	switch status {
	case MachineAgentAuthStatusUnknown, MachineAgentAuthStatusLoggedIn, MachineAgentAuthStatusNotLoggedIn:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported auth status %q", strings.TrimSpace(raw))
	}
}

func parseMachineAgentAuthMode(raw string) (MachineAgentAuthMode, error) {
	mode := MachineAgentAuthMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case MachineAgentAuthModeUnknown, MachineAgentAuthModeLogin, MachineAgentAuthModeAPIKey:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported auth mode %q", strings.TrimSpace(raw))
	}
}

func kilobytesToGigabytes(value float64) float64 {
	return roundTwoDecimals(value / (1024.0 * 1024.0))
}

func percentage(part float64, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundTwoDecimals((part / total) * 100)
}

func roundTwoDecimals(value float64) float64 {
	const factor = 100
	return math.Round(value*factor) / factor
}
