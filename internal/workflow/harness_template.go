package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

type BuildHarnessTemplateDataInput = domain.BuildHarnessTemplateDataInput

type HarnessTemplateData domain.HarnessTemplateData

type HarnessTicketData = domain.HarnessTicketData

type HarnessTicketLinkData = domain.HarnessTicketLinkData

type HarnessTicketDependencyData = domain.HarnessTicketDependencyData

type HarnessProjectData = domain.HarnessProjectData

type HarnessProjectWorkflowData = domain.HarnessProjectWorkflowData

type HarnessProjectWorkflowTicketData = domain.HarnessProjectWorkflowTicketData

type HarnessProjectStatusData = domain.HarnessProjectStatusData

type HarnessProjectMachineData = domain.HarnessProjectMachineData

type HarnessProjectUpdateThreadData = domain.HarnessProjectUpdateThreadData

type HarnessProjectUpdateCommentData = domain.HarnessProjectUpdateCommentData

type HarnessRepoData = domain.HarnessRepoData

type HarnessAgentData = domain.HarnessAgentData

type HarnessMachineData = domain.HarnessMachineData

type HarnessAccessibleMachineData = domain.HarnessAccessibleMachineData

type HarnessWorkflowData = domain.HarnessWorkflowData

type HarnessPlatformData = domain.HarnessPlatformData

type HarnessVariableGroup = domain.HarnessVariableGroup

type HarnessVariableMetadata = domain.HarnessVariableMetadata

func init() {
	if !gonja.DefaultEnvironment.Filters.Exists("markdown_escape") {
		_ = gonja.DefaultEnvironment.Filters.Register("markdown_escape", filterMarkdownEscape)
	}
}

func RenderHarnessBody(content string, data HarnessTemplateData) (string, error) {
	normalized := normalizeHarnessNewlines(content)
	if strings.TrimSpace(normalized) == "" {
		return "", nil
	}
	if err := validateHarnessForSave(normalized); err != nil {
		return "", err
	}

	template, err := gonja.FromString(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	rendered, err := template.ExecuteToString(exec.NewContext(data.contextMap()))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	return rendered, nil
}

func HarnessVariableDictionary() []HarnessVariableGroup {
	return []HarnessVariableGroup{
		{
			Name: "ticket",
			Variables: []HarnessVariableMetadata{
				{Path: "ticket.id", Type: "string", Description: "Ticket UUID", Example: "550e8400-e29b-41d4-a716-446655440000"},
				{Path: "ticket.identifier", Type: "string", Description: "Human-readable ticket identifier", Example: "ASE-42"},
				{Path: "ticket.title", Type: "string", Description: "Ticket title", Example: "Fix login form validation"},
				{Path: "ticket.description", Type: "string", Description: "Ticket description markdown", Example: "The login form does not validate..."},
				{Path: "ticket.status", Type: "string", Description: "Current status name", Example: "In Progress"},
				{Path: "ticket.priority", Type: "string", Description: "Priority", Example: "high"},
				{Path: "ticket.type", Type: "string", Description: "Ticket type", Example: "bugfix"},
				{Path: "ticket.created_by", Type: "string", Description: "Created by", Example: "user:gary"},
				{Path: "ticket.created_at", Type: "string", Description: "Created-at time in ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "ticket.attempt_count", Type: "int", Description: "Current attempt count", Example: "1"},
				{Path: "ticket.max_attempts", Type: "int", Description: "Maximum attempt count", Example: "3"},
				{Path: "ticket.budget_usd", Type: "float", Description: "Budget cap (USD)", Example: "5.00"},
				{Path: "ticket.external_ref", Type: "string", Description: "External reference identifier", Example: "octocat/repo#42"},
				{Path: "ticket.parent_identifier", Type: "string", Description: "Parent ticket identifier", Example: "ASE-30"},
				{Path: "ticket.url", Type: "string", Description: "OpenASE Web UI ticket URL", Example: "http://localhost:19836/tickets/ASE-42"},
				{Path: "ticket.links", Type: "list", Description: "External link list"},
				{Path: "ticket.links[].type", Type: "string", Description: "External link type", Example: "github_issue"},
				{Path: "ticket.links[].url", Type: "string", Description: "External URL", Example: "https://github.com/acme/backend/issues/42"},
				{Path: "ticket.links[].title", Type: "string", Description: "External title", Example: "Login validation broken on Safari"},
				{Path: "ticket.links[].status", Type: "string", Description: "External status", Example: "open"},
				{Path: "ticket.links[].relation", Type: "string", Description: "Relation type", Example: "resolves"},
				{Path: "ticket.dependencies", Type: "list", Description: "Dependent ticket list"},
				{Path: "ticket.dependencies[].identifier", Type: "string", Description: "Dependent ticket identifier", Example: "ASE-30"},
				{Path: "ticket.dependencies[].title", Type: "string", Description: "Dependent ticket title", Example: "Design user auth schema"},
				{Path: "ticket.dependencies[].type", Type: "string", Description: "Dependency type", Example: "blocks"},
				{Path: "ticket.dependencies[].status", Type: "string", Description: "Dependent ticket status", Example: "Done"},
			},
		},
		{
			Name: "project",
			Variables: []HarnessVariableMetadata{
				{Path: "project.id", Type: "string", Description: "Project UUID"},
				{Path: "project.name", Type: "string", Description: "Project name", Example: "awesome-saas"},
				{Path: "project.slug", Type: "string", Description: "Project slug", Example: "awesome-saas"},
				{Path: "project.description", Type: "string", Description: "Project description", Example: "A SaaS platform for..."},
				{Path: "project.status", Type: "string", Description: "Project status", Example: "In Progress"},
				{Path: "project.workflows", Type: "list", Description: "List of active workflows on the project"},
				{Path: "project.workflows[].name", Type: "string", Description: "Workflow name", Example: "Coding Workflow"},
				{Path: "project.workflows[].type", Type: "string", Description: "Workflow type", Example: "coding"},
				{Path: "project.workflows[].role_name", Type: "string", Description: "Role name", Example: "fullstack-developer"},
				{Path: "project.workflows[].role_description", Type: "string", Description: "Role description", Example: "Implement product changes end to end, covering backend, frontend, and verification."},
				{Path: "project.workflows[].pickup_status", Type: "string", Description: "Workflow pickup status", Example: "Todo"},
				{Path: "project.workflows[].finish_status", Type: "string", Description: "Workflow finish status", Example: "Done"},
				{Path: "project.workflows[].pickup_statuses", Type: "list", Description: "Structured list of workflow pickup statuses"},
				{Path: "project.workflows[].pickup_statuses[].id", Type: "string", Description: "Status UUID"},
				{Path: "project.workflows[].pickup_statuses[].name", Type: "string", Description: "Status name", Example: "Todo"},
				{Path: "project.workflows[].pickup_statuses[].stage", Type: "string", Description: "Status stage", Example: "unstarted"},
				{Path: "project.workflows[].pickup_statuses[].color", Type: "string", Description: "Status color", Example: "#3B82F6"},
				{Path: "project.workflows[].finish_statuses", Type: "list", Description: "Structured list of workflow finish statuses"},
				{Path: "project.workflows[].finish_statuses[].id", Type: "string", Description: "Status UUID"},
				{Path: "project.workflows[].finish_statuses[].name", Type: "string", Description: "Status name", Example: "Done"},
				{Path: "project.workflows[].finish_statuses[].stage", Type: "string", Description: "Status stage", Example: "completed"},
				{Path: "project.workflows[].finish_statuses[].color", Type: "string", Description: "Status color", Example: "#10B981"},
				{Path: "project.workflows[].harness_path", Type: "string", Description: "Workflow harness file path", Example: ".openase/harnesses/coding.md"},
				{Path: "project.workflows[].harness_content", Type: "string", Description: "Current workflow harness content"},
				{Path: "project.workflows[].skills", Type: "list", Description: "Skills currently bound to the workflow harness"},
				{Path: "project.workflows[].max_concurrent", Type: "int", Description: "Maximum concurrency", Example: "3"},
				{Path: "project.workflows[].current_active", Type: "int", Description: "Current active ticket count", Example: "1"},
				{Path: "project.workflows[].recent_tickets", Type: "list", Description: "Recent ticket history for this workflow"},
				{Path: "project.workflows[].recent_tickets[].identifier", Type: "string", Description: "Ticket identifier", Example: "ASE-40"},
				{Path: "project.workflows[].recent_tickets[].title", Type: "string", Description: "Ticket title", Example: "Implement auth boundary parsing"},
				{Path: "project.workflows[].recent_tickets[].status", Type: "string", Description: "Current status", Example: "Done"},
				{Path: "project.workflows[].recent_tickets[].priority", Type: "string", Description: "Priority", Example: "high"},
				{Path: "project.workflows[].recent_tickets[].type", Type: "string", Description: "Ticket type", Example: "bugfix"},
				{Path: "project.workflows[].recent_tickets[].attempt_count", Type: "int", Description: "Attempt count", Example: "2"},
				{Path: "project.workflows[].recent_tickets[].consecutive_errors", Type: "int", Description: "Consecutive failure count", Example: "1"},
				{Path: "project.workflows[].recent_tickets[].retry_paused", Type: "bool", Description: "Whether retries are paused", Example: "false"},
				{Path: "project.workflows[].recent_tickets[].pause_reason", Type: "string", Description: "Pause reason", Example: "budget_exhausted"},
				{Path: "project.workflows[].recent_tickets[].created_at", Type: "string", Description: "Created-at time in ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "project.workflows[].recent_tickets[].started_at", Type: "string", Description: "Execution start time in ISO 8601", Example: "2026-03-19T10:40:00Z"},
				{Path: "project.workflows[].recent_tickets[].completed_at", Type: "string", Description: "Completion time in ISO 8601", Example: "2026-03-19T10:52:00Z"},
				{Path: "project.statuses", Type: "list", Description: "Project status list"},
				{Path: "project.statuses[].id", Type: "string", Description: "Status UUID"},
				{Path: "project.statuses[].name", Type: "string", Description: "Status name", Example: "Backlog"},
				{Path: "project.statuses[].stage", Type: "string", Description: "Status stage", Example: "backlog"},
				{Path: "project.statuses[].color", Type: "string", Description: "Status color", Example: "#6B7280"},
				{Path: "project.machines", Type: "list", Description: "Project-accessible machine view"},
				{Path: "project.machines[].name", Type: "string", Description: "Machine name", Example: "gpu-01"},
				{Path: "project.machines[].host", Type: "string", Description: "Machine host", Example: "10.0.1.10"},
				{Path: "project.machines[].description", Type: "string", Description: "Machine description", Example: "NVIDIA A100 x4"},
				{Path: "project.machines[].labels", Type: "list", Description: "Machine labels", Example: "[\"gpu\", \"a100\"]"},
				{Path: "project.machines[].status", Type: "string", Description: "Machine availability status", Example: "current"},
				{Path: "project.machines[].resources", Type: "object", Description: "Most recent persisted machine resource snapshot; empty if no probe has run yet"},
				{Path: "project.updates", Type: "list", Description: "Project-level curated updates timeline, ordered by most recent activity first"},
				{Path: "project.updates[].id", Type: "string", Description: "Update thread UUID"},
				{Path: "project.updates[].status", Type: "string", Description: "Progress status", Example: "at_risk"},
				{Path: "project.updates[].title", Type: "string", Description: "Update title", Example: "Release train checkpoint"},
				{Path: "project.updates[].body_markdown", Type: "string", Description: "Update body markdown"},
				{Path: "project.updates[].created_by", Type: "string", Description: "Update author", Example: "agent:dispatcher-01"},
				{Path: "project.updates[].created_at", Type: "string", Description: "Created-at time in ISO 8601"},
				{Path: "project.updates[].updated_at", Type: "string", Description: "Updated-at time in ISO 8601"},
				{Path: "project.updates[].last_activity_at", Type: "string", Description: "Last activity time in ISO 8601"},
				{Path: "project.updates[].comment_count", Type: "int", Description: "Undeleted comment count", Example: "2"},
				{Path: "project.updates[].comments", Type: "list", Description: "Comments under this update"},
				{Path: "project.updates[].comments[].id", Type: "string", Description: "Comment UUID"},
				{Path: "project.updates[].comments[].body_markdown", Type: "string", Description: "Comment body markdown"},
				{Path: "project.updates[].comments[].created_by", Type: "string", Description: "Comment author"},
				{Path: "project.updates[].comments[].created_at", Type: "string", Description: "Comment created-at time in ISO 8601"},
				{Path: "project.updates[].comments[].updated_at", Type: "string", Description: "Comment updated-at time in ISO 8601"},
			},
		},
		{
			Name: "repos",
			Variables: []HarnessVariableMetadata{
				{Path: "repos", Type: "list", Description: "Repositories scoped to the current ticket"},
				{Path: "repos[].name", Type: "string", Description: "Repository alias", Example: "backend"},
				{Path: "repos[].url", Type: "string", Description: "Repository URL", Example: "https://github.com/acme/backend"},
				{Path: "repos[].path", Type: "string", Description: "Workspace-local path", Example: "/workspaces/ASE-42/backend"},
				{Path: "repos[].branch", Type: "string", Description: "Current working branch", Example: "agent/ASE-42"},
				{Path: "repos[].default_branch", Type: "string", Description: "Repository default branch", Example: "main"},
				{Path: "repos[].labels", Type: "list", Description: "Repository labels", Example: "[\"go\", \"backend\", \"api\"]"},
				{Path: "all_repos", Type: "list", Description: "All repositories under the project"},
			},
		},
		{
			Name: "agent",
			Variables: []HarnessVariableMetadata{
				{Path: "agent.id", Type: "string", Description: "Agent UUID"},
				{Path: "agent.name", Type: "string", Description: "Agent name", Example: "claude-01"},
				{Path: "agent.provider", Type: "string", Description: "Provider name", Example: "Claude Code"},
				{Path: "agent.adapter_type", Type: "string", Description: "Adapter type", Example: "claude-code-cli"},
				{Path: "agent.model", Type: "string", Description: "Model name", Example: "claude-sonnet-4-6"},
				{Path: "agent.total_tickets_completed", Type: "int", Description: "Historical completed ticket count", Example: "47"},
			},
		},
		{
			Name: "machine",
			Variables: []HarnessVariableMetadata{
				{Path: "machine.name", Type: "string", Description: "Current execution machine name", Example: "gpu-01"},
				{Path: "machine.host", Type: "string", Description: "Machine host", Example: "10.0.1.10"},
				{Path: "machine.description", Type: "string", Description: "Machine description", Example: "NVIDIA A100 x4"},
				{Path: "machine.labels", Type: "list", Description: "Machine labels", Example: "[\"gpu\", \"a100\"]"},
				{Path: "machine.workspace_root", Type: "string", Description: "Remote workspace root", Example: "/home/openase/workspaces"},
				{Path: "accessible_machines", Type: "list", Description: "Accessible machine list"},
				{Path: "accessible_machines[].name", Type: "string", Description: "Machine name", Example: "storage"},
				{Path: "accessible_machines[].host", Type: "string", Description: "Machine host", Example: "10.0.1.20"},
				{Path: "accessible_machines[].description", Type: "string", Description: "Machine description", Example: "Data storage, 16TB NVMe"},
				{Path: "accessible_machines[].labels", Type: "list", Description: "Machine labels", Example: "[\"storage\", \"nfs\"]"},
				{Path: "accessible_machines[].ssh_user", Type: "string", Description: "SSH username", Example: "openase"},
			},
		},
		{
			Name: "context",
			Variables: []HarnessVariableMetadata{
				{Path: "attempt", Type: "int", Description: "Current attempt count (starting from 1)", Example: "1"},
				{Path: "max_attempts", Type: "int", Description: "Maximum attempt count", Example: "3"},
				{Path: "workspace", Type: "string", Description: "Workspace root path", Example: "/workspaces/ASE-42"},
				{Path: "timestamp", Type: "string", Description: "Current time in ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "openase_version", Type: "string", Description: "OpenASE version", Example: "0.3.1"},
			},
		},
		{
			Name: "workflow",
			Variables: []HarnessVariableMetadata{
				{Path: "workflow.name", Type: "string", Description: "Workflow name", Example: "coding"},
				{Path: "workflow.type", Type: "string", Description: "Workflow type", Example: "coding"},
				{Path: "workflow.role_name", Type: "string", Description: "Role name", Example: "fullstack-developer"},
				{Path: "workflow.pickup_status", Type: "string", Description: "Pickup status", Example: "Todo"},
				{Path: "workflow.finish_status", Type: "string", Description: "Finish status", Example: "Done"},
			},
		},
		{
			Name: "platform",
			Variables: []HarnessVariableMetadata{
				{Path: "platform.api_url", Type: "string", Description: "Platform API URL", Example: "http://localhost:19836/api/v1"},
				{Path: "platform.agent_token", Type: "string", Description: "Short-lived agent token", Example: "ase_agent_xxx"},
				{Path: "platform.project_id", Type: "string", Description: "Current project ID"},
				{Path: "platform.ticket_id", Type: "string", Description: "Current ticket ID"},
			},
		},
		{
			Name: "filters",
			Variables: []HarnessVariableMetadata{
				{Path: "default(value)", Type: "filter", Description: "Use a default value when the variable is empty"},
				{Path: "truncate(length)", Type: "filter", Description: "Truncate to the specified length"},
				{Path: "join(sep)", Type: "filter", Description: "Join a list into a string"},
				{Path: "upper", Type: "filter", Description: "Convert to uppercase"},
				{Path: "lower", Type: "filter", Description: "Convert to lowercase"},
				{Path: "length", Type: "filter", Description: "Get length"},
				{Path: "first", Type: "filter", Description: "Get the first element"},
				{Path: "last", Type: "filter", Description: "Get the last element"},
				{Path: "sort", Type: "filter", Description: "Sort"},
				{Path: "selectattr(attr, value)", Type: "filter", Description: "Filter a list by attribute"},
				{Path: "map(attribute)", Type: "filter", Description: "Extract an attribute list"},
				{Path: "tojson", Type: "filter", Description: "Render a JSON string"},
				{Path: "markdown_escape", Type: "filter", Description: "Escape Markdown special characters"},
			},
		},
	}
}

func (d HarnessTemplateData) contextMap() map[string]any {
	return map[string]any{
		"ticket": map[string]any{
			"id":                d.Ticket.ID,
			"identifier":        d.Ticket.Identifier,
			"title":             d.Ticket.Title,
			"description":       d.Ticket.Description,
			"status":            d.Ticket.Status,
			"priority":          d.Ticket.Priority,
			"type":              d.Ticket.Type,
			"created_by":        d.Ticket.CreatedBy,
			"created_at":        d.Ticket.CreatedAt,
			"attempt_count":     d.Ticket.AttemptCount,
			"max_attempts":      d.Ticket.MaxAttempts,
			"budget_usd":        d.Ticket.BudgetUSD,
			"external_ref":      d.Ticket.ExternalRef,
			"parent_identifier": d.Ticket.ParentIdentifier,
			"url":               d.Ticket.URL,
			"links":             ticketLinkMaps(d.Ticket.Links),
			"dependencies":      dependencyMaps(d.Ticket.Dependencies),
		},
		"project": map[string]any{
			"id":          d.Project.ID,
			"name":        d.Project.Name,
			"slug":        d.Project.Slug,
			"description": d.Project.Description,
			"status":      d.Project.Status,
			"workflows":   projectWorkflowMaps(d.Project.Workflows),
			"statuses":    projectStatusMaps(d.Project.Statuses),
			"machines":    projectMachineMaps(d.Project.Machines),
			"updates":     projectUpdateThreadMaps(d.Project.Updates),
		},
		"repos":               repoMaps(d.Repos),
		"all_repos":           repoMaps(d.AllRepos),
		"agent":               agentMap(d.Agent),
		"machine":             machineMap(d.Machine),
		"accessible_machines": accessibleMachineMaps(d.AccessibleMachines),
		"attempt":             d.Attempt,
		"max_attempts":        d.MaxAttempts,
		"workspace":           d.Workspace,
		"timestamp":           d.Timestamp,
		"openase_version":     d.OpenASEVersion,
		"workflow": map[string]any{
			"name":          d.Workflow.Name,
			"type":          d.Workflow.Type,
			"role_name":     d.Workflow.RoleName,
			"pickup_status": d.Workflow.PickupStatus,
			"finish_status": d.Workflow.FinishStatus,
		},
		"platform": map[string]any{
			"api_url":     d.Platform.APIURL,
			"agent_token": d.Platform.AgentToken,
			"project_id":  d.Platform.ProjectID,
			"ticket_id":   d.Platform.TicketID,
		},
	}
}

func mapHarnessProjectMachines(current HarnessMachineData, accessible []HarnessAccessibleMachineData) []HarnessProjectMachineData {
	items := make([]HarnessProjectMachineData, 0, len(accessible)+1)
	seen := make(map[string]struct{}, len(accessible)+1)
	add := func(name string, host string, description string, labels []string, status string, resources map[string]any) {
		key := strings.TrimSpace(name) + "|" + strings.TrimSpace(host)
		if key == "|" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		items = append(items, HarnessProjectMachineData{
			Name:        name,
			Host:        host,
			Description: description,
			Labels:      append([]string(nil), labels...),
			Status:      status,
			Resources:   cloneResourceMap(resources),
		})
	}

	add(current.Name, current.Host, current.Description, current.Labels, "current", current.Resources)
	for _, machine := range accessible {
		add(machine.Name, machine.Host, machine.Description, machine.Labels, "accessible", machine.Resources)
	}
	slices.SortFunc(items, func(left, right HarnessProjectMachineData) int {
		if compared := strings.Compare(left.Name, right.Name); compared != 0 {
			return compared
		}
		return strings.Compare(left.Host, right.Host)
	})

	return items
}

func normalizeAttemptCount(raw int) int {
	if raw < 1 {
		return 1
	}
	return raw
}

func normalizePlatformData(input HarnessPlatformData, projectID uuid.UUID, ticketID uuid.UUID) HarnessPlatformData {
	data := input
	if strings.TrimSpace(data.ProjectID) == "" {
		data.ProjectID = projectID.String()
	}
	if strings.TrimSpace(data.TicketID) == "" {
		data.TicketID = ticketID.String()
	}
	return data
}

func resolveRepoPath(workspaceDirname string, workspace string, repoName string) string {
	if trimmed := strings.TrimSpace(workspaceDirname); trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(workspace, repoName))
}

func cloneHarnessMachine(machine HarnessMachineData) HarnessMachineData {
	machine.Labels = append([]string(nil), machine.Labels...)
	return machine
}

func cloneAccessibleMachines(machines []HarnessAccessibleMachineData) []HarnessAccessibleMachineData {
	items := make([]HarnessAccessibleMachineData, 0, len(machines))
	for _, machine := range machines {
		cloned := machine
		cloned.Labels = append([]string(nil), machine.Labels...)
		items = append(items, cloned)
	}
	return items
}

func ticketLinkMaps(items []HarnessTicketLinkData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"type":     item.Type,
			"url":      item.URL,
			"title":    item.Title,
			"status":   item.Status,
			"relation": item.Relation,
		})
	}
	return result
}

func dependencyMaps(items []HarnessTicketDependencyData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"identifier": item.Identifier,
			"title":      item.Title,
			"type":       item.Type,
			"status":     item.Status,
		})
	}
	return result
}

func repoMaps(items []HarnessRepoData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"name":           item.Name,
			"url":            item.URL,
			"path":           item.Path,
			"branch":         item.Branch,
			"default_branch": item.DefaultBranch,
			"labels":         append([]string(nil), item.Labels...),
		})
	}
	return result
}

func projectWorkflowMaps(items []HarnessProjectWorkflowData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"name":             item.Name,
			"type":             item.Type,
			"role_name":        item.RoleName,
			"role_description": item.RoleDescription,
			"pickup_status":    item.PickupStatus,
			"finish_status":    item.FinishStatus,
			"pickup_statuses":  projectStatusMaps(item.PickupStatuses),
			"finish_statuses":  projectStatusMaps(item.FinishStatuses),
			"harness_path":     item.HarnessPath,
			"harness_content":  item.HarnessContent,
			"skills":           append([]string(nil), item.Skills...),
			"max_concurrent":   item.MaxConcurrent,
			"current_active":   item.CurrentActive,
			"recent_tickets":   projectWorkflowTicketMaps(item.RecentTickets),
		})
	}
	return result
}

func projectWorkflowTicketMaps(items []HarnessProjectWorkflowTicketData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"identifier":         item.Identifier,
			"title":              item.Title,
			"status":             item.Status,
			"priority":           item.Priority,
			"type":               item.Type,
			"attempt_count":      item.AttemptCount,
			"consecutive_errors": item.ConsecutiveErrors,
			"retry_paused":       item.RetryPaused,
			"pause_reason":       item.PauseReason,
			"created_at":         item.CreatedAt,
			"started_at":         item.StartedAt,
			"completed_at":       item.CompletedAt,
		})
	}
	return result
}

func projectStatusMaps(items []HarnessProjectStatusData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"id":    item.ID,
			"name":  item.Name,
			"stage": item.Stage,
			"color": item.Color,
		})
	}
	return result
}

func projectUpdateThreadMaps(items []HarnessProjectUpdateThreadData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"id":               item.ID,
			"status":           item.Status,
			"title":            item.Title,
			"body_markdown":    item.BodyMarkdown,
			"created_by":       item.CreatedBy,
			"created_at":       item.CreatedAt,
			"updated_at":       item.UpdatedAt,
			"last_activity_at": item.LastActivityAt,
			"comment_count":    item.CommentCount,
			"comments":         projectUpdateCommentMaps(item.Comments),
		})
	}
	return result
}

func projectUpdateCommentMaps(items []HarnessProjectUpdateCommentData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"id":            item.ID,
			"body_markdown": item.BodyMarkdown,
			"created_by":    item.CreatedBy,
			"created_at":    item.CreatedAt,
			"updated_at":    item.UpdatedAt,
		})
	}
	return result
}

func projectMachineMaps(items []HarnessProjectMachineData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"name":        item.Name,
			"host":        item.Host,
			"description": item.Description,
			"labels":      append([]string(nil), item.Labels...),
			"status":      item.Status,
			"resources":   cloneResourceMap(item.Resources),
		})
	}
	return result
}

func agentMap(item HarnessAgentData) map[string]any {
	return map[string]any{
		"id":                      item.ID,
		"name":                    item.Name,
		"provider":                item.Provider,
		"adapter_type":            item.AdapterType,
		"model":                   item.Model,
		"total_tickets_completed": item.TotalTicketsCompleted,
	}
}

func machineMap(item HarnessMachineData) map[string]any {
	return map[string]any{
		"name":           item.Name,
		"host":           item.Host,
		"description":    item.Description,
		"labels":         append([]string(nil), item.Labels...),
		"resources":      cloneResourceMap(item.Resources),
		"workspace_root": item.WorkspaceRoot,
	}
}

func accessibleMachineMaps(items []HarnessAccessibleMachineData) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"name":        item.Name,
			"host":        item.Host,
			"description": item.Description,
			"labels":      append([]string(nil), item.Labels...),
			"resources":   cloneResourceMap(item.Resources),
			"ssh_user":    item.SSHUser,
		})
	}
	return result
}

func cloneResourceMap(resources map[string]any) map[string]any {
	if len(resources) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(resources))
	for key, value := range resources {
		cloned[key] = value
	}
	return cloned
}

func formatOptionalTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func filterMarkdownEscape(_ *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
	if in.IsError() {
		return in
	}
	if p := params.ExpectNothing(); p.IsError() {
		return exec.AsValue(fmt.Errorf("wrong signature for 'markdown_escape': %v", p))
	}

	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
		">", "\\>",
		"|", "\\|",
	)

	return exec.AsValue(replacer.Replace(in.String()))
}
