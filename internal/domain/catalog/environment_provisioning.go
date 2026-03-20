package catalog

import (
	"fmt"
	"strings"
)

const (
	EnvironmentProvisionerRoleSlug            = "env-provisioner"
	EnvironmentProvisionerRoleName            = "Environment Provisioner"
	EnvironmentProvisionerSkillInstallClaude  = "install-claude-code"
	EnvironmentProvisionerSkillInstallCodex   = "install-codex"
	EnvironmentProvisionerSkillSetupGit       = "setup-git"
	EnvironmentProvisionerSkillSetupGitHubCLI = "setup-gh-cli"
)

type MachineEnvironmentProvisioningIssue struct {
	Code      string  `json:"code"`
	Source    string  `json:"source"`
	Title     string  `json:"title"`
	Detail    string  `json:"detail"`
	SkillName *string `json:"skill_name,omitempty"`
}

type MachineEnvironmentProvisioningPlan struct {
	Available         bool                                  `json:"available"`
	Needed            bool                                  `json:"needed"`
	Runnable          bool                                  `json:"runnable"`
	RoleSlug          string                                `json:"role_slug"`
	RoleName          string                                `json:"role_name"`
	RequiredSkills    []string                              `json:"required_skills"`
	Summary           string                                `json:"summary"`
	Issues            []MachineEnvironmentProvisioningIssue `json:"issues"`
	Notes             []string                              `json:"notes"`
	TicketTitle       string                                `json:"ticket_title"`
	TicketDescription string                                `json:"ticket_description"`
}

type storedAgentEnvironment struct {
	ClaudeCode storedCLI
	Codex      storedCLI
}

type storedCLI struct {
	Installed  bool
	AuthStatus MachineAgentAuthStatus
}

type storedFullAudit struct {
	Git     storedGitAudit
	GitHub  storedGitHubCLIAudit
	Network storedNetworkAudit
}

type storedGitAudit struct {
	Installed bool
	UserName  string
	UserEmail string
}

type storedGitHubCLIAudit struct {
	Installed  bool
	AuthStatus MachineAgentAuthStatus
}

type storedNetworkAudit struct {
	GitHubReachable bool
	PyPIReachable   bool
	NPMReachable    bool
}

func PlanMachineEnvironmentProvisioning(machine Machine) MachineEnvironmentProvisioningPlan {
	plan := MachineEnvironmentProvisioningPlan{
		RoleSlug:       EnvironmentProvisionerRoleSlug,
		RoleName:       EnvironmentProvisionerRoleName,
		RequiredSkills: []string{},
		Issues:         []MachineEnvironmentProvisioningIssue{},
		Notes:          []string{},
		TicketTitle:    fmt.Sprintf("Provision environment on %s", machine.Name),
	}

	agentEnvironment, agentErr := parseStoredAgentEnvironment(machine.Resources)
	fullAudit, auditErr := parseStoredFullAudit(machine.Resources)
	if agentErr != nil {
		plan.Notes = append(plan.Notes, fmt.Sprintf("L4 agent environment snapshot is unavailable: %v", agentErr))
	}
	if auditErr != nil {
		plan.Notes = append(plan.Notes, fmt.Sprintf("L5 full audit snapshot is unavailable: %v", auditErr))
	}
	plan.Available = agentErr == nil || auditErr == nil

	if agentErr == nil {
		plan.appendCLIIssues(agentEnvironment)
	}
	if auditErr == nil {
		plan.appendAuditIssues(fullAudit)
		plan.appendNetworkNotes(fullAudit.Network)
	}

	plan.Needed = len(plan.Issues) > 0
	plan.Runnable = plan.Needed && machine.Host != LocalMachineHost && machine.Status != "offline" && machineIsReachable(machine.Resources)

	switch {
	case !plan.Available:
		plan.Summary = fmt.Sprintf("Machine monitor L4/L5 snapshots are not available for %s yet.", machine.Name)
	case !plan.Needed:
		plan.Summary = fmt.Sprintf("No environment remediation actions are currently recommended for %s.", machine.Name)
	case plan.Runnable:
		plan.Summary = fmt.Sprintf("%d environment remediation actions are ready for %s.", len(plan.Issues), machine.Name)
	default:
		plan.Summary = fmt.Sprintf("%d remediation actions were identified for %s, but SSH execution is not currently available.", len(plan.Issues), machine.Name)
	}

	if machine.Host == LocalMachineHost {
		plan.Notes = append(plan.Notes, "The environment provisioner role targets SSH-managed machines; the local control-plane host is not a remote provisioner target.")
	} else if machine.Status == "offline" || !machineIsReachable(machine.Resources) {
		plan.Notes = append(plan.Notes, "Machine reachability is degraded, so the SSH agent runner cannot execute the provisioner role until L1 connectivity recovers.")
	}

	plan.TicketDescription = buildProvisioningTicketDescription(machine, plan)
	return plan
}

func (p *MachineEnvironmentProvisioningPlan) appendCLIIssues(environment storedAgentEnvironment) {
	p.appendCLIIssue("claude_code", "Claude Code", environment.ClaudeCode, EnvironmentProvisionerSkillInstallClaude)
	p.appendCLIIssue("codex", "Codex CLI", environment.Codex, EnvironmentProvisionerSkillInstallCodex)
}

func (p *MachineEnvironmentProvisioningPlan) appendCLIIssue(code string, displayName string, cli storedCLI, skillName string) {
	switch {
	case !cli.Installed:
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      code + "_missing",
			Source:    "l4",
			Title:     displayName + " is not installed",
			Detail:    fmt.Sprintf("Machine Monitor L4 reported %s installed=false.", code),
			SkillName: stringPointer(skillName),
		})
	case cli.AuthStatus == MachineAgentAuthStatusNotLoggedIn:
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      code + "_auth",
			Source:    "l4",
			Title:     displayName + " is installed but not authenticated",
			Detail:    fmt.Sprintf("Machine Monitor L4 reported %s auth_status=%q.", code, cli.AuthStatus),
			SkillName: stringPointer(skillName),
		})
	}
}

func (p *MachineEnvironmentProvisioningPlan) appendAuditIssues(audit storedFullAudit) {
	switch {
	case !audit.Git.Installed:
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      "git_missing",
			Source:    "l5",
			Title:     "Git is not installed",
			Detail:    "Machine Monitor L5 reported git installed=false.",
			SkillName: stringPointer(EnvironmentProvisionerSkillSetupGit),
		})
	case audit.Git.UserName == "" || audit.Git.UserEmail == "":
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      "git_identity_missing",
			Source:    "l5",
			Title:     "Git identity is incomplete",
			Detail:    "Machine Monitor L5 reported an empty git user.name or user.email.",
			SkillName: stringPointer(EnvironmentProvisionerSkillSetupGit),
		})
	}

	switch {
	case !audit.GitHub.Installed:
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      "gh_cli_missing",
			Source:    "l5",
			Title:     "GitHub CLI is not installed",
			Detail:    "Machine Monitor L5 reported gh CLI installed=false.",
			SkillName: stringPointer(EnvironmentProvisionerSkillSetupGitHubCLI),
		})
	case audit.GitHub.AuthStatus != MachineAgentAuthStatusLoggedIn:
		p.appendIssue(MachineEnvironmentProvisioningIssue{
			Code:      "gh_cli_auth",
			Source:    "l5",
			Title:     "GitHub CLI is installed but not authenticated",
			Detail:    fmt.Sprintf("Machine Monitor L5 reported gh CLI auth_status=%q.", audit.GitHub.AuthStatus),
			SkillName: stringPointer(EnvironmentProvisionerSkillSetupGitHubCLI),
		})
	}
}

func (p *MachineEnvironmentProvisioningPlan) appendNetworkNotes(network storedNetworkAudit) {
	requiredSkillSet := make(map[string]struct{}, len(p.RequiredSkills))
	for _, skill := range p.RequiredSkills {
		requiredSkillSet[skill] = struct{}{}
	}

	if !network.GitHubReachable {
		p.appendNote("GitHub is unreachable from the machine; git and gh bootstrap steps may fail until github.com connectivity is restored.")
	}
	if !network.NPMReachable {
		if _, ok := requiredSkillSet[EnvironmentProvisionerSkillInstallClaude]; ok {
			p.appendNote("The npm registry is unreachable; Claude Code installation may fail until registry.npmjs.org connectivity is restored.")
		}
	}
	if !network.PyPIReachable {
		if _, ok := requiredSkillSet[EnvironmentProvisionerSkillInstallCodex]; ok {
			p.appendNote("PyPI is unreachable; Codex installation may fail until pypi.org connectivity is restored.")
		}
	}
}

func (p *MachineEnvironmentProvisioningPlan) appendIssue(issue MachineEnvironmentProvisioningIssue) {
	p.Issues = append(p.Issues, issue)
	if issue.SkillName == nil {
		return
	}
	for _, existing := range p.RequiredSkills {
		if existing == *issue.SkillName {
			return
		}
	}
	p.RequiredSkills = append(p.RequiredSkills, *issue.SkillName)
}

func (p *MachineEnvironmentProvisioningPlan) appendNote(note string) {
	for _, existing := range p.Notes {
		if existing == note {
			return
		}
	}
	p.Notes = append(p.Notes, note)
}

func buildProvisioningTicketDescription(machine Machine, plan MachineEnvironmentProvisioningPlan) string {
	var builder strings.Builder
	builder.WriteString("Environment Provisioner should repair the target machine.\n\n")
	builder.WriteString("Target machine:\n")
	_, _ = fmt.Fprintf(&builder, "- Name: %s\n", machine.Name)
	_, _ = fmt.Fprintf(&builder, "- Host: %s\n", machine.Host)
	_, _ = fmt.Fprintf(&builder, "- Status: %s\n", machine.Status)
	_, _ = fmt.Fprintf(&builder, "- Runnable over SSH: %t\n", plan.Runnable)
	builder.WriteString("\nDetected issues:\n")
	if len(plan.Issues) == 0 {
		builder.WriteString("- None.\n")
	} else {
		for _, issue := range plan.Issues {
			builder.WriteString("- ")
			builder.WriteString(issue.Title)
			if issue.SkillName != nil {
				builder.WriteString(" via `")
				builder.WriteString(*issue.SkillName)
				builder.WriteString("`")
			}
			builder.WriteString(". ")
			builder.WriteString(issue.Detail)
			builder.WriteString("\n")
		}
	}
	if len(plan.Notes) > 0 {
		builder.WriteString("\nNotes:\n")
		for _, note := range plan.Notes {
			builder.WriteString("- ")
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}
	if len(plan.RequiredSkills) > 0 {
		builder.WriteString("\nRequired skills:\n")
		for _, skill := range plan.RequiredSkills {
			builder.WriteString("- ")
			builder.WriteString(skill)
			builder.WriteString("\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

func parseStoredAgentEnvironment(resources map[string]any) (storedAgentEnvironment, error) {
	raw, ok := nestedObject(resources, "agent_environment")
	if !ok {
		return storedAgentEnvironment{}, fmt.Errorf("missing agent_environment")
	}

	claudeRaw, ok := nestedObject(raw, "claude_code")
	if !ok {
		return storedAgentEnvironment{}, fmt.Errorf("missing agent_environment.claude_code")
	}
	codexRaw, ok := nestedObject(raw, "codex")
	if !ok {
		return storedAgentEnvironment{}, fmt.Errorf("missing agent_environment.codex")
	}

	claude, err := parseStoredCLI(claudeRaw)
	if err != nil {
		return storedAgentEnvironment{}, fmt.Errorf("parse claude_code: %w", err)
	}
	codex, err := parseStoredCLI(codexRaw)
	if err != nil {
		return storedAgentEnvironment{}, fmt.Errorf("parse codex: %w", err)
	}

	return storedAgentEnvironment{
		ClaudeCode: claude,
		Codex:      codex,
	}, nil
}

func parseStoredCLI(raw map[string]any) (storedCLI, error) {
	installed, ok := boolField(raw, "installed")
	if !ok {
		return storedCLI{}, fmt.Errorf("missing installed")
	}
	authStatusRaw, ok := stringField(raw, "auth_status")
	if !ok {
		return storedCLI{}, fmt.Errorf("missing auth_status")
	}
	authStatus, err := parseMachineAgentAuthStatus(authStatusRaw)
	if err != nil {
		return storedCLI{}, fmt.Errorf("parse auth_status: %w", err)
	}

	return storedCLI{
		Installed:  installed,
		AuthStatus: authStatus,
	}, nil
}

func parseStoredFullAudit(resources map[string]any) (storedFullAudit, error) {
	raw, ok := nestedObject(resources, "full_audit")
	if !ok {
		return storedFullAudit{}, fmt.Errorf("missing full_audit")
	}

	gitRaw, ok := nestedObject(raw, "git")
	if !ok {
		return storedFullAudit{}, fmt.Errorf("missing full_audit.git")
	}
	ghRaw, ok := nestedObject(raw, "gh_cli")
	if !ok {
		return storedFullAudit{}, fmt.Errorf("missing full_audit.gh_cli")
	}
	networkRaw, ok := nestedObject(raw, "network")
	if !ok {
		return storedFullAudit{}, fmt.Errorf("missing full_audit.network")
	}

	gitAudit, err := parseStoredGitAudit(gitRaw)
	if err != nil {
		return storedFullAudit{}, fmt.Errorf("parse git audit: %w", err)
	}
	ghAudit, err := parseStoredGitHubCLIAudit(ghRaw)
	if err != nil {
		return storedFullAudit{}, fmt.Errorf("parse gh audit: %w", err)
	}
	networkAudit, err := parseStoredNetworkAudit(networkRaw)
	if err != nil {
		return storedFullAudit{}, fmt.Errorf("parse network audit: %w", err)
	}

	return storedFullAudit{
		Git:     gitAudit,
		GitHub:  ghAudit,
		Network: networkAudit,
	}, nil
}

func parseStoredGitAudit(raw map[string]any) (storedGitAudit, error) {
	installed, ok := boolField(raw, "installed")
	if !ok {
		return storedGitAudit{}, fmt.Errorf("missing installed")
	}

	return storedGitAudit{
		Installed: installed,
		UserName:  stringFieldOrEmpty(raw, "user_name"),
		UserEmail: stringFieldOrEmpty(raw, "user_email"),
	}, nil
}

func parseStoredGitHubCLIAudit(raw map[string]any) (storedGitHubCLIAudit, error) {
	installed, ok := boolField(raw, "installed")
	if !ok {
		return storedGitHubCLIAudit{}, fmt.Errorf("missing installed")
	}
	authStatusRaw, ok := stringField(raw, "auth_status")
	if !ok {
		return storedGitHubCLIAudit{}, fmt.Errorf("missing auth_status")
	}
	authStatus, err := parseMachineAgentAuthStatus(authStatusRaw)
	if err != nil {
		return storedGitHubCLIAudit{}, fmt.Errorf("parse auth_status: %w", err)
	}

	return storedGitHubCLIAudit{
		Installed:  installed,
		AuthStatus: authStatus,
	}, nil
}

func parseStoredNetworkAudit(raw map[string]any) (storedNetworkAudit, error) {
	githubReachable, ok := boolField(raw, "github_reachable")
	if !ok {
		return storedNetworkAudit{}, fmt.Errorf("missing github_reachable")
	}
	pypiReachable, ok := boolField(raw, "pypi_reachable")
	if !ok {
		return storedNetworkAudit{}, fmt.Errorf("missing pypi_reachable")
	}
	npmReachable, ok := boolField(raw, "npm_reachable")
	if !ok {
		return storedNetworkAudit{}, fmt.Errorf("missing npm_reachable")
	}

	return storedNetworkAudit{
		GitHubReachable: githubReachable,
		PyPIReachable:   pypiReachable,
		NPMReachable:    npmReachable,
	}, nil
}

func machineIsReachable(resources map[string]any) bool {
	monitor, ok := nestedObject(resources, "monitor")
	if ok {
		l1, ok := nestedObject(monitor, "l1")
		if ok {
			reachable, ok := boolField(l1, "reachable")
			if ok {
				return reachable
			}
		}
	}
	reachable, ok := boolField(resources, "last_success")
	return ok && reachable
}

func nestedObject(raw map[string]any, key string) (map[string]any, bool) {
	value, ok := raw[key]
	if !ok {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func boolField(raw map[string]any, key string) (bool, bool) {
	value, ok := raw[key]
	if !ok {
		return false, false
	}
	typed, ok := value.(bool)
	return typed, ok
}

func stringField(raw map[string]any, key string) (string, bool) {
	value, ok := raw[key]
	if !ok {
		return "", false
	}
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	return typed, true
}

func stringFieldOrEmpty(raw map[string]any, key string) string {
	value, _ := stringField(raw, key)
	return value
}

func stringPointer(value string) *string {
	copied := value
	return &copied
}
