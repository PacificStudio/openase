package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/google/uuid"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"go.yaml.in/yaml/v3"
)

type BuildHarnessTemplateDataInput struct {
	WorkflowID         uuid.UUID
	TicketID           uuid.UUID
	AgentID            *uuid.UUID
	Workspace          string
	Timestamp          time.Time
	OpenASEVersion     string
	TicketURL          string
	Platform           HarnessPlatformData
	Machine            HarnessMachineData
	AccessibleMachines []HarnessAccessibleMachineData
}

type HarnessTemplateData struct {
	Ticket             HarnessTicketData
	Project            HarnessProjectData
	Repos              []HarnessRepoData
	AllRepos           []HarnessRepoData
	Agent              HarnessAgentData
	Machine            HarnessMachineData
	AccessibleMachines []HarnessAccessibleMachineData
	Attempt            int
	MaxAttempts        int
	Workspace          string
	Timestamp          string
	OpenASEVersion     string
	Workflow           HarnessWorkflowData
	Platform           HarnessPlatformData
}

type HarnessTicketData struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	Status           string
	Priority         string
	Type             string
	CreatedBy        string
	CreatedAt        string
	AttemptCount     int
	MaxAttempts      int
	BudgetUSD        float64
	ExternalRef      string
	ParentIdentifier string
	URL              string
	Links            []HarnessTicketLinkData
	Dependencies     []HarnessTicketDependencyData
}

type HarnessTicketLinkData struct {
	Type     string
	URL      string
	Title    string
	Status   string
	Relation string
}

type HarnessTicketDependencyData struct {
	Identifier string
	Title      string
	Type       string
	Status     string
}

type HarnessProjectData struct {
	ID            string
	Name          string
	Slug          string
	Description   string
	Status        string
	DefaultBranch string
	Workflows     []HarnessProjectWorkflowData
	Statuses      []HarnessProjectStatusData
	Machines      []HarnessProjectMachineData
}

type HarnessProjectWorkflowData struct {
	Name            string
	Type            string
	RoleName        string
	RoleDescription string
	PickupStatus    string
	FinishStatus    string
	HarnessPath     string
	HarnessContent  string
	Skills          []string
	MaxConcurrent   int
	CurrentActive   int
	RecentTickets   []HarnessProjectWorkflowTicketData
}

type HarnessProjectWorkflowTicketData struct {
	Identifier        string
	Title             string
	Status            string
	Priority          string
	Type              string
	AttemptCount      int
	ConsecutiveErrors int
	RetryPaused       bool
	PauseReason       string
	CreatedAt         string
	StartedAt         string
	CompletedAt       string
}

type HarnessProjectStatusData struct {
	Name  string
	Color string
}

type HarnessProjectMachineData struct {
	Name        string
	Host        string
	Description string
	Labels      []string
	Status      string
	Resources   map[string]any
}

type HarnessRepoData struct {
	Name          string
	URL           string
	Path          string
	Branch        string
	DefaultBranch string
	Labels        []string
	IsPrimary     bool
}

type HarnessAgentData struct {
	ID                    string
	Name                  string
	Provider              string
	AdapterType           string
	Model                 string
	Capabilities          []string
	TotalTicketsCompleted int
}

type HarnessMachineData struct {
	Name          string
	Host          string
	Description   string
	Labels        []string
	WorkspaceRoot string
}

type HarnessAccessibleMachineData struct {
	Name        string
	Host        string
	Description string
	Labels      []string
	SSHUser     string
}

type HarnessWorkflowData struct {
	Name         string
	Type         string
	RoleName     string
	PickupStatus string
	FinishStatus string
}

type HarnessPlatformData struct {
	APIURL     string
	AgentToken string
	ProjectID  string
	TicketID   string
}

type HarnessVariableGroup struct {
	Name      string                    `json:"name"`
	Variables []HarnessVariableMetadata `json:"variables"`
}

type HarnessVariableMetadata struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

func init() {
	if !gonja.DefaultEnvironment.Filters.Exists("markdown_escape") {
		_ = gonja.DefaultEnvironment.Filters.Register("markdown_escape", filterMarkdownEscape)
	}
}

func (s *Service) BuildHarnessTemplateData(ctx context.Context, input BuildHarnessTemplateDataInput) (HarnessTemplateData, error) {
	if s == nil || s.client == nil {
		return HarnessTemplateData{}, ErrUnavailable
	}

	workflowItem, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(input.WorkflowID)).
		WithProject().
		WithPickupStatus().
		WithFinishStatus().
		Only(ctx)
	if err != nil {
		return HarnessTemplateData{}, s.mapWorkflowReadError("get workflow for harness render", err)
	}

	ticketItem, err := s.client.Ticket.Query().
		Where(entticket.IDEQ(input.TicketID)).
		WithStatus().
		WithParent().
		WithExternalLinks().
		WithRepoScopes(func(query *ent.TicketRepoScopeQuery) {
			query.WithRepo()
		}).
		WithOutgoingDependencies(func(query *ent.TicketDependencyQuery) {
			query.WithTargetTicket(func(target *ent.TicketQuery) {
				target.WithStatus()
			})
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return HarnessTemplateData{}, ErrWorkflowNotFound
		}
		return HarnessTemplateData{}, fmt.Errorf("get ticket for harness render: %w", err)
	}

	if ticketItem.ProjectID != workflowItem.ProjectID {
		return HarnessTemplateData{}, fmt.Errorf("%w: workflow and ticket belong to different projects", ErrHarnessInvalid)
	}

	projectItem, err := s.client.Project.Query().
		Where(entproject.IDEQ(workflowItem.ProjectID)).
		WithRepos(func(query *ent.ProjectRepoQuery) {
			query.Order(ent.Asc(entprojectrepo.FieldName))
		}).
		WithWorkflows(func(query *ent.WorkflowQuery) {
			query.
				Order(ent.Asc(entworkflow.FieldName)).
				WithPickupStatus().
				WithFinishStatus()
		}).
		WithStatuses(func(query *ent.TicketStatusQuery) {
			query.
				Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return HarnessTemplateData{}, ErrProjectNotFound
		}
		return HarnessTemplateData{}, fmt.Errorf("get project for harness render: %w", err)
	}

	agentData := HarnessAgentData{}
	if input.AgentID != nil {
		agentItem, agentErr := s.client.Agent.Query().
			Where(
				entagent.IDEQ(*input.AgentID),
				entagent.ProjectIDEQ(workflowItem.ProjectID),
			).
			WithProvider().
			Only(ctx)
		if agentErr != nil {
			if ent.IsNotFound(agentErr) {
				return HarnessTemplateData{}, fmt.Errorf("%w: agent not found for workflow project", ErrHarnessInvalid)
			}
			return HarnessTemplateData{}, fmt.Errorf("get agent for harness render: %w", agentErr)
		}
		agentData = mapHarnessAgent(agentItem)
	}

	harnessContent, err := s.registry.Read(workflowItem.HarnessPath)
	if err != nil {
		return HarnessTemplateData{}, fmt.Errorf("read workflow harness for role extraction: %w", err)
	}

	attemptCount := normalizeAttemptCount(ticketItem.AttemptCount)
	maxAttempts := max(workflowItem.MaxRetryAttempts, 0)
	workspace := strings.TrimSpace(input.Workspace)
	renderTime := input.Timestamp.UTC()
	if renderTime.IsZero() {
		renderTime = time.Now().UTC()
	}

	scopedRepos, repoBranchByID := mapHarnessScopedRepos(ticketItem.Edges.RepoScopes, workspace)
	allRepos := mapHarnessAllRepos(projectItem.Edges.Repos, repoBranchByID, workspace)
	defaultBranch := deriveDefaultBranch(projectItem.Edges.Repos)
	projectWorkflows, err := s.mapHarnessProjectWorkflows(ctx, projectItem.Edges.Workflows)
	if err != nil {
		return HarnessTemplateData{}, err
	}
	finishStatus := ""
	if workflowItem.Edges.FinishStatus != nil {
		finishStatus = workflowItem.Edges.FinishStatus.Name
	}

	data := HarnessTemplateData{
		Ticket: HarnessTicketData{
			ID:               ticketItem.ID.String(),
			Identifier:       ticketItem.Identifier,
			Title:            ticketItem.Title,
			Description:      ticketItem.Description,
			Status:           edgeTicketStatusName(ticketItem.Edges.Status),
			Priority:         ticketItem.Priority.String(),
			Type:             ticketItem.Type.String(),
			CreatedBy:        ticketItem.CreatedBy,
			CreatedAt:        ticketItem.CreatedAt.UTC().Format(time.RFC3339),
			AttemptCount:     attemptCount,
			MaxAttempts:      maxAttempts,
			BudgetUSD:        ticketItem.BudgetUsd,
			ExternalRef:      ticketItem.ExternalRef,
			ParentIdentifier: parentIdentifier(ticketItem),
			URL:              strings.TrimSpace(input.TicketURL),
			Links:            mapHarnessTicketLinks(ticketItem.Edges.ExternalLinks),
			Dependencies:     mapHarnessDependencies(ticketItem.Edges.OutgoingDependencies),
		},
		Project: HarnessProjectData{
			ID:            projectItem.ID.String(),
			Name:          projectItem.Name,
			Slug:          projectItem.Slug,
			Description:   projectItem.Description,
			Status:        projectItem.Status.String(),
			DefaultBranch: defaultBranch,
			Workflows:     projectWorkflows,
			Statuses:      mapHarnessProjectStatuses(projectItem.Edges.Statuses),
			Machines:      mapHarnessProjectMachines(input.Machine, input.AccessibleMachines),
		},
		Repos:              scopedRepos,
		AllRepos:           allRepos,
		Agent:              agentData,
		Machine:            cloneHarnessMachine(input.Machine),
		AccessibleMachines: cloneAccessibleMachines(input.AccessibleMachines),
		Attempt:            attemptCount,
		MaxAttempts:        maxAttempts,
		Workspace:          workspace,
		Timestamp:          renderTime.Format(time.RFC3339),
		OpenASEVersion:     strings.TrimSpace(input.OpenASEVersion),
		Workflow: HarnessWorkflowData{
			Name:         workflowItem.Name,
			Type:         workflowItem.Type.String(),
			RoleName:     extractWorkflowRoleName(harnessContent, workflowItem.Name),
			PickupStatus: edgeTicketStatusName(workflowItem.Edges.PickupStatus),
			FinishStatus: finishStatus,
		},
		Platform: normalizePlatformData(input.Platform, workflowItem.ProjectID, ticketItem.ID),
	}

	return data, nil
}

func RenderHarnessBody(content string, data HarnessTemplateData) (string, error) {
	_, body, err := extractHarnessFrontmatter(content)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}
	if strings.TrimSpace(body) == "" {
		return "", nil
	}

	template, err := gonja.FromString(body)
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
				{Path: "ticket.id", Type: "string", Description: "工单 UUID", Example: "550e8400-e29b-41d4-a716-446655440000"},
				{Path: "ticket.identifier", Type: "string", Description: "可读工单标识", Example: "ASE-42"},
				{Path: "ticket.title", Type: "string", Description: "工单标题", Example: "Fix login form validation"},
				{Path: "ticket.description", Type: "string", Description: "工单描述 Markdown", Example: "The login form does not validate..."},
				{Path: "ticket.status", Type: "string", Description: "当前状态名", Example: "In Progress"},
				{Path: "ticket.priority", Type: "string", Description: "优先级", Example: "high"},
				{Path: "ticket.type", Type: "string", Description: "工单类型", Example: "bugfix"},
				{Path: "ticket.created_by", Type: "string", Description: "创建者", Example: "user:gary"},
				{Path: "ticket.created_at", Type: "string", Description: "创建时间 ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "ticket.attempt_count", Type: "int", Description: "当前尝试次数", Example: "1"},
				{Path: "ticket.max_attempts", Type: "int", Description: "最大尝试次数", Example: "3"},
				{Path: "ticket.budget_usd", Type: "float", Description: "预算上限（美元）", Example: "5.00"},
				{Path: "ticket.external_ref", Type: "string", Description: "外部关联标识", Example: "octocat/repo#42"},
				{Path: "ticket.parent_identifier", Type: "string", Description: "父工单标识", Example: "ASE-30"},
				{Path: "ticket.url", Type: "string", Description: "OpenASE Web UI 工单链接", Example: "http://localhost:19836/tickets/ASE-42"},
				{Path: "ticket.links", Type: "list", Description: "外部链接列表"},
				{Path: "ticket.links[].type", Type: "string", Description: "外部链接类型", Example: "github_issue"},
				{Path: "ticket.links[].url", Type: "string", Description: "外部 URL", Example: "https://github.com/acme/backend/issues/42"},
				{Path: "ticket.links[].title", Type: "string", Description: "外部标题", Example: "Login validation broken on Safari"},
				{Path: "ticket.links[].status", Type: "string", Description: "外部状态", Example: "open"},
				{Path: "ticket.links[].relation", Type: "string", Description: "关联关系", Example: "resolves"},
				{Path: "ticket.dependencies", Type: "list", Description: "依赖工单列表"},
				{Path: "ticket.dependencies[].identifier", Type: "string", Description: "依赖工单标识", Example: "ASE-30"},
				{Path: "ticket.dependencies[].title", Type: "string", Description: "依赖工单标题", Example: "Design user auth schema"},
				{Path: "ticket.dependencies[].type", Type: "string", Description: "依赖类型", Example: "blocks"},
				{Path: "ticket.dependencies[].status", Type: "string", Description: "依赖工单状态", Example: "Done"},
			},
		},
		{
			Name: "project",
			Variables: []HarnessVariableMetadata{
				{Path: "project.id", Type: "string", Description: "项目 UUID"},
				{Path: "project.name", Type: "string", Description: "项目名称", Example: "awesome-saas"},
				{Path: "project.slug", Type: "string", Description: "项目 slug", Example: "awesome-saas"},
				{Path: "project.description", Type: "string", Description: "项目描述", Example: "A SaaS platform for..."},
				{Path: "project.status", Type: "string", Description: "项目状态", Example: "active"},
				{Path: "project.default_branch", Type: "string", Description: "项目默认分支", Example: "main"},
				{Path: "project.workflows", Type: "list", Description: "项目中已激活的 Workflow 列表"},
				{Path: "project.workflows[].name", Type: "string", Description: "Workflow 名称", Example: "Coding Workflow"},
				{Path: "project.workflows[].type", Type: "string", Description: "Workflow 类型", Example: "coding"},
				{Path: "project.workflows[].role_name", Type: "string", Description: "角色名称", Example: "fullstack-developer"},
				{Path: "project.workflows[].role_description", Type: "string", Description: "角色描述", Example: "Implement product changes end to end, covering backend, frontend, and verification."},
				{Path: "project.workflows[].pickup_status", Type: "string", Description: "Workflow 的 pickup 状态", Example: "Todo"},
				{Path: "project.workflows[].finish_status", Type: "string", Description: "Workflow 的 finish 状态", Example: "Done"},
				{Path: "project.workflows[].harness_path", Type: "string", Description: "Workflow Harness 文件路径", Example: ".openase/harnesses/coding.md"},
				{Path: "project.workflows[].harness_content", Type: "string", Description: "Workflow 当前 Harness 内容"},
				{Path: "project.workflows[].skills", Type: "list", Description: "当前 Workflow Harness 绑定的技能列表"},
				{Path: "project.workflows[].max_concurrent", Type: "int", Description: "最大并发数", Example: "3"},
				{Path: "project.workflows[].current_active", Type: "int", Description: "当前活跃工单数", Example: "1"},
				{Path: "project.workflows[].recent_tickets", Type: "list", Description: "最近使用该 Workflow 的工单历史"},
				{Path: "project.workflows[].recent_tickets[].identifier", Type: "string", Description: "工单标识", Example: "ASE-40"},
				{Path: "project.workflows[].recent_tickets[].title", Type: "string", Description: "工单标题", Example: "Implement auth boundary parsing"},
				{Path: "project.workflows[].recent_tickets[].status", Type: "string", Description: "当前状态", Example: "Done"},
				{Path: "project.workflows[].recent_tickets[].priority", Type: "string", Description: "优先级", Example: "high"},
				{Path: "project.workflows[].recent_tickets[].type", Type: "string", Description: "工单类型", Example: "bugfix"},
				{Path: "project.workflows[].recent_tickets[].attempt_count", Type: "int", Description: "尝试次数", Example: "2"},
				{Path: "project.workflows[].recent_tickets[].consecutive_errors", Type: "int", Description: "连续失败次数", Example: "1"},
				{Path: "project.workflows[].recent_tickets[].retry_paused", Type: "bool", Description: "是否已暂停重试", Example: "false"},
				{Path: "project.workflows[].recent_tickets[].pause_reason", Type: "string", Description: "暂停原因", Example: "budget_exhausted"},
				{Path: "project.workflows[].recent_tickets[].created_at", Type: "string", Description: "创建时间 ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "project.workflows[].recent_tickets[].started_at", Type: "string", Description: "开始执行时间 ISO 8601", Example: "2026-03-19T10:40:00Z"},
				{Path: "project.workflows[].recent_tickets[].completed_at", Type: "string", Description: "完成时间 ISO 8601", Example: "2026-03-19T10:52:00Z"},
				{Path: "project.statuses", Type: "list", Description: "项目状态列表"},
				{Path: "project.statuses[].name", Type: "string", Description: "状态名", Example: "Backlog"},
				{Path: "project.statuses[].color", Type: "string", Description: "状态颜色", Example: "#6B7280"},
				{Path: "project.machines", Type: "list", Description: "项目可访问的机器视图"},
				{Path: "project.machines[].name", Type: "string", Description: "机器名", Example: "gpu-01"},
				{Path: "project.machines[].host", Type: "string", Description: "机器地址", Example: "10.0.1.10"},
				{Path: "project.machines[].description", Type: "string", Description: "机器描述", Example: "NVIDIA A100 x4"},
				{Path: "project.machines[].labels", Type: "list", Description: "机器标签", Example: "[\"gpu\", \"a100\"]"},
				{Path: "project.machines[].status", Type: "string", Description: "机器可用状态", Example: "current"},
				{Path: "project.machines[].resources", Type: "object", Description: "资源快照（当前最小实现为空对象）"},
			},
		},
		{
			Name: "repos",
			Variables: []HarnessVariableMetadata{
				{Path: "repos", Type: "list", Description: "当前工单涉及的仓库"},
				{Path: "repos[].name", Type: "string", Description: "仓库别名", Example: "backend"},
				{Path: "repos[].url", Type: "string", Description: "仓库 URL", Example: "https://github.com/acme/backend"},
				{Path: "repos[].path", Type: "string", Description: "工作区本地路径", Example: "/workspaces/ASE-42/backend"},
				{Path: "repos[].branch", Type: "string", Description: "当前工作分支", Example: "agent/claude-01/ASE-42"},
				{Path: "repos[].default_branch", Type: "string", Description: "仓库默认分支", Example: "main"},
				{Path: "repos[].labels", Type: "list", Description: "仓库标签", Example: "[\"go\", \"backend\", \"api\"]"},
				{Path: "repos[].is_primary", Type: "bool", Description: "是否主仓库", Example: "true"},
				{Path: "all_repos", Type: "list", Description: "项目下的全部仓库"},
			},
		},
		{
			Name: "agent",
			Variables: []HarnessVariableMetadata{
				{Path: "agent.id", Type: "string", Description: "Agent UUID"},
				{Path: "agent.name", Type: "string", Description: "Agent 名称", Example: "claude-01"},
				{Path: "agent.provider", Type: "string", Description: "Provider 名称", Example: "Claude Code"},
				{Path: "agent.adapter_type", Type: "string", Description: "适配器类型", Example: "claude-code-cli"},
				{Path: "agent.model", Type: "string", Description: "模型名称", Example: "claude-sonnet-4-6"},
				{Path: "agent.capabilities", Type: "list", Description: "能力标签", Example: "[\"backend\", \"go\", \"testing\"]"},
				{Path: "agent.total_tickets_completed", Type: "int", Description: "历史完成工单数", Example: "47"},
			},
		},
		{
			Name: "machine",
			Variables: []HarnessVariableMetadata{
				{Path: "machine.name", Type: "string", Description: "当前执行机器名", Example: "gpu-01"},
				{Path: "machine.host", Type: "string", Description: "机器地址", Example: "10.0.1.10"},
				{Path: "machine.description", Type: "string", Description: "机器描述", Example: "NVIDIA A100 x4"},
				{Path: "machine.labels", Type: "list", Description: "机器标签", Example: "[\"gpu\", \"a100\"]"},
				{Path: "machine.workspace_root", Type: "string", Description: "远端工作区根目录", Example: "/home/openase/workspaces"},
				{Path: "accessible_machines", Type: "list", Description: "可访问机器列表"},
				{Path: "accessible_machines[].name", Type: "string", Description: "机器名", Example: "storage"},
				{Path: "accessible_machines[].host", Type: "string", Description: "机器地址", Example: "10.0.1.20"},
				{Path: "accessible_machines[].description", Type: "string", Description: "机器描述", Example: "数据存储, 16TB NVMe"},
				{Path: "accessible_machines[].labels", Type: "list", Description: "机器标签", Example: "[\"storage\", \"nfs\"]"},
				{Path: "accessible_machines[].ssh_user", Type: "string", Description: "SSH 用户名", Example: "openase"},
			},
		},
		{
			Name: "context",
			Variables: []HarnessVariableMetadata{
				{Path: "attempt", Type: "int", Description: "当前尝试次数（从 1 开始）", Example: "1"},
				{Path: "max_attempts", Type: "int", Description: "最大尝试次数", Example: "3"},
				{Path: "workspace", Type: "string", Description: "工作区根路径", Example: "/workspaces/ASE-42"},
				{Path: "timestamp", Type: "string", Description: "当前时间 ISO 8601", Example: "2026-03-19T10:30:00Z"},
				{Path: "openase_version", Type: "string", Description: "OpenASE 版本号", Example: "0.3.1"},
			},
		},
		{
			Name: "workflow",
			Variables: []HarnessVariableMetadata{
				{Path: "workflow.name", Type: "string", Description: "Workflow 名称", Example: "coding"},
				{Path: "workflow.type", Type: "string", Description: "Workflow 类型", Example: "coding"},
				{Path: "workflow.role_name", Type: "string", Description: "角色名称", Example: "fullstack-developer"},
				{Path: "workflow.pickup_status", Type: "string", Description: "Pickup 状态", Example: "Todo"},
				{Path: "workflow.finish_status", Type: "string", Description: "Finish 状态", Example: "Done"},
			},
		},
		{
			Name: "platform",
			Variables: []HarnessVariableMetadata{
				{Path: "platform.api_url", Type: "string", Description: "Platform API 地址", Example: "http://localhost:19836/api/v1"},
				{Path: "platform.agent_token", Type: "string", Description: "Agent 短期 Token", Example: "ase_agent_xxx"},
				{Path: "platform.project_id", Type: "string", Description: "当前项目 ID"},
				{Path: "platform.ticket_id", Type: "string", Description: "当前工单 ID"},
			},
		},
		{
			Name: "filters",
			Variables: []HarnessVariableMetadata{
				{Path: "default(value)", Type: "filter", Description: "变量为空时使用默认值"},
				{Path: "truncate(length)", Type: "filter", Description: "截断到指定长度"},
				{Path: "join(sep)", Type: "filter", Description: "列表拼接为字符串"},
				{Path: "upper", Type: "filter", Description: "转大写"},
				{Path: "lower", Type: "filter", Description: "转小写"},
				{Path: "length", Type: "filter", Description: "获取长度"},
				{Path: "first", Type: "filter", Description: "获取第一个元素"},
				{Path: "last", Type: "filter", Description: "获取最后一个元素"},
				{Path: "sort", Type: "filter", Description: "排序"},
				{Path: "selectattr(attr, value)", Type: "filter", Description: "按属性过滤列表"},
				{Path: "map(attribute)", Type: "filter", Description: "提取属性列表"},
				{Path: "tojson", Type: "filter", Description: "输出 JSON 字符串"},
				{Path: "markdown_escape", Type: "filter", Description: "转义 Markdown 特殊字符"},
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
			"id":             d.Project.ID,
			"name":           d.Project.Name,
			"slug":           d.Project.Slug,
			"description":    d.Project.Description,
			"status":         d.Project.Status,
			"default_branch": d.Project.DefaultBranch,
			"workflows":      projectWorkflowMaps(d.Project.Workflows),
			"statuses":       projectStatusMaps(d.Project.Statuses),
			"machines":       projectMachineMaps(d.Project.Machines),
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

func mapHarnessScopedRepos(scopes []*ent.TicketRepoScope, workspace string) ([]HarnessRepoData, map[uuid.UUID]string) {
	repos := make([]HarnessRepoData, 0, len(scopes))
	branches := make(map[uuid.UUID]string, len(scopes))
	for _, scope := range scopes {
		repo := scope.Edges.Repo
		if repo == nil {
			continue
		}
		branches[repo.ID] = scope.BranchName
		repos = append(repos, HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.ClonePath, workspace, repo.Name),
			Branch:        scope.BranchName,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
			IsPrimary:     repo.IsPrimary,
		})
	}
	return repos, branches
}

func mapHarnessAllRepos(repos []*ent.ProjectRepo, repoBranchByID map[uuid.UUID]string, workspace string) []HarnessRepoData {
	items := make([]HarnessRepoData, 0, len(repos))
	for _, repo := range repos {
		branch := repoBranchByID[repo.ID]
		if branch == "" {
			branch = repo.DefaultBranch
		}
		items = append(items, HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.ClonePath, workspace, repo.Name),
			Branch:        branch,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
			IsPrimary:     repo.IsPrimary,
		})
	}
	return items
}

func (s *Service) mapHarnessProjectWorkflows(ctx context.Context, workflows []*ent.Workflow) ([]HarnessProjectWorkflowData, error) {
	items := make([]HarnessProjectWorkflowData, 0, len(workflows))
	workflowIDs := make([]uuid.UUID, 0, len(workflows))
	for _, workflowItem := range workflows {
		if workflowItem == nil || !workflowItem.IsActive {
			continue
		}
		workflowIDs = append(workflowIDs, workflowItem.ID)
	}

	activeCountByWorkflow := make(map[uuid.UUID]int, len(workflowIDs))
	if len(workflowIDs) > 0 {
		activeTickets, err := s.client.Ticket.Query().
			Where(
				entticket.WorkflowIDIn(workflowIDs...),
				entticket.AssignedAgentIDNotNil(),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list active workflow tickets for harness render: %w", err)
		}
		for _, ticketItem := range activeTickets {
			if ticketItem.WorkflowID == nil {
				continue
			}
			activeCountByWorkflow[*ticketItem.WorkflowID]++
		}
	}

	for _, workflowItem := range workflows {
		if workflowItem == nil || !workflowItem.IsActive {
			continue
		}
		harnessContent, err := s.registry.Read(workflowItem.HarnessPath)
		if err != nil {
			return nil, fmt.Errorf("read workflow harness for project context: %w", err)
		}
		roleName := extractWorkflowRoleName(harnessContent, workflowItem.Name)
		skills, err := ParseHarnessSkills(harnessContent)
		if err != nil {
			return nil, fmt.Errorf("parse workflow skills for project context: %w", err)
		}
		recentTickets, err := s.listHarnessWorkflowRecentTickets(ctx, workflowItem.ID, 5)
		if err != nil {
			return nil, err
		}
		finishStatus := ""
		if workflowItem.Edges.FinishStatus != nil {
			finishStatus = workflowItem.Edges.FinishStatus.Name
		}
		items = append(items, HarnessProjectWorkflowData{
			Name:            workflowItem.Name,
			Type:            workflowItem.Type.String(),
			RoleName:        roleName,
			RoleDescription: extractWorkflowRoleDescription(harnessContent),
			PickupStatus:    edgeTicketStatusName(workflowItem.Edges.PickupStatus),
			FinishStatus:    finishStatus,
			HarnessPath:     workflowItem.HarnessPath,
			HarnessContent:  harnessContent,
			Skills:          skills,
			MaxConcurrent:   workflowItem.MaxConcurrent,
			CurrentActive:   activeCountByWorkflow[workflowItem.ID],
			RecentTickets:   recentTickets,
		})
	}

	return items, nil
}

func (s *Service) listHarnessWorkflowRecentTickets(ctx context.Context, workflowID uuid.UUID, limit int) ([]HarnessProjectWorkflowTicketData, error) {
	query := s.client.Ticket.Query().
		Where(entticket.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entticket.FieldCreatedAt)).
		WithStatus()
	if limit > 0 {
		query = query.Limit(limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow history for harness render: %w", err)
	}

	tickets := make([]HarnessProjectWorkflowTicketData, 0, len(items))
	for _, item := range items {
		tickets = append(tickets, mapHarnessProjectWorkflowTicket(item))
	}
	return tickets, nil
}

func mapHarnessProjectStatuses(statuses []*ent.TicketStatus) []HarnessProjectStatusData {
	items := make([]HarnessProjectStatusData, 0, len(statuses))
	for _, status := range statuses {
		if status == nil {
			continue
		}
		items = append(items, HarnessProjectStatusData{
			Name:  status.Name,
			Color: status.Color,
		})
	}
	return items
}

func mapHarnessProjectMachines(current HarnessMachineData, accessible []HarnessAccessibleMachineData) []HarnessProjectMachineData {
	items := make([]HarnessProjectMachineData, 0, len(accessible)+1)
	seen := make(map[string]struct{}, len(accessible)+1)
	add := func(name string, host string, description string, labels []string, status string) {
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
			Resources:   map[string]any{},
		})
	}

	add(current.Name, current.Host, current.Description, current.Labels, "current")
	for _, machine := range accessible {
		add(machine.Name, machine.Host, machine.Description, machine.Labels, "accessible")
	}
	slices.SortFunc(items, func(left, right HarnessProjectMachineData) int {
		if compared := strings.Compare(left.Name, right.Name); compared != 0 {
			return compared
		}
		return strings.Compare(left.Host, right.Host)
	})

	return items
}

func mapHarnessAgent(item *ent.Agent) HarnessAgentData {
	if item == nil {
		return HarnessAgentData{}
	}

	providerName := ""
	adapterType := ""
	modelName := ""
	if item.Edges.Provider != nil {
		providerName = item.Edges.Provider.Name
		adapterType = item.Edges.Provider.AdapterType.String()
		modelName = item.Edges.Provider.ModelName
	}

	return HarnessAgentData{
		ID:                    item.ID.String(),
		Name:                  item.Name,
		Provider:              providerName,
		AdapterType:           adapterType,
		Model:                 modelName,
		Capabilities:          append([]string(nil), item.Capabilities...),
		TotalTicketsCompleted: item.TotalTicketsCompleted,
	}
}

func mapHarnessTicketLinks(links []*ent.TicketExternalLink) []HarnessTicketLinkData {
	items := make([]HarnessTicketLinkData, 0, len(links))
	for _, link := range links {
		items = append(items, HarnessTicketLinkData{
			Type:     link.LinkType.String(),
			URL:      link.URL,
			Title:    link.Title,
			Status:   link.Status,
			Relation: link.Relation.String(),
		})
	}
	return items
}

func mapHarnessDependencies(dependencies []*ent.TicketDependency) []HarnessTicketDependencyData {
	items := make([]HarnessTicketDependencyData, 0, len(dependencies))
	for _, dependency := range dependencies {
		target := dependency.Edges.TargetTicket
		if target == nil {
			continue
		}
		items = append(items, HarnessTicketDependencyData{
			Identifier: target.Identifier,
			Title:      target.Title,
			Type:       normalizeDependencyType(dependency.Type),
			Status:     edgeTicketStatusName(target.Edges.Status),
		})
	}
	return items
}

func edgeTicketStatusName(status *ent.TicketStatus) string {
	if status == nil {
		return ""
	}
	return status.Name
}

func parentIdentifier(ticketItem *ent.Ticket) string {
	if ticketItem == nil || ticketItem.Edges.Parent == nil {
		return ""
	}
	return ticketItem.Edges.Parent.Identifier
}

func normalizeDependencyType(value entticketdependency.Type) string {
	return strings.ReplaceAll(value.String(), "-", "_")
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

func deriveDefaultBranch(repos []*ent.ProjectRepo) string {
	for _, repo := range repos {
		if repo.IsPrimary && strings.TrimSpace(repo.DefaultBranch) != "" {
			return repo.DefaultBranch
		}
	}
	for _, repo := range repos {
		if strings.TrimSpace(repo.DefaultBranch) != "" {
			return repo.DefaultBranch
		}
	}
	return "main"
}

func resolveRepoPath(clonePath string, workspace string, repoName string) string {
	if trimmed := strings.TrimSpace(clonePath); trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(workspace, repoName))
}

func extractWorkflowRoleName(content string, fallback string) string {
	frontmatter, _, err := extractHarnessFrontmatter(content)
	if err != nil {
		return fallback
	}

	var document struct {
		Workflow map[string]any `yaml:"workflow"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return fallback
	}

	for _, key := range []string{"role_name", "role", "name"} {
		value, ok := document.Workflow[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}

	return fallback
}

func extractWorkflowRoleDescription(content string) string {
	_, body, err := extractHarnessFrontmatter(content)
	if err != nil {
		return ""
	}

	var paragraph []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			if len(paragraph) > 0 {
				return strings.Join(paragraph, " ")
			}
		case strings.HasPrefix(trimmed, "#"):
			continue
		default:
			paragraph = append(paragraph, trimmed)
		}
	}

	return strings.Join(paragraph, " ")
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
			"is_primary":     item.IsPrimary,
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
			"name":  item.Name,
			"color": item.Color,
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
		"capabilities":            append([]string(nil), item.Capabilities...),
		"total_tickets_completed": item.TotalTicketsCompleted,
	}
}

func machineMap(item HarnessMachineData) map[string]any {
	return map[string]any{
		"name":           item.Name,
		"host":           item.Host,
		"description":    item.Description,
		"labels":         append([]string(nil), item.Labels...),
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
			"ssh_user":    item.SSHUser,
		})
	}
	return result
}

func mapHarnessProjectWorkflowTicket(item *ent.Ticket) HarnessProjectWorkflowTicketData {
	return HarnessProjectWorkflowTicketData{
		Identifier:        item.Identifier,
		Title:             item.Title,
		Status:            edgeTicketStatusName(item.Edges.Status),
		Priority:          item.Priority.String(),
		Type:              item.Type.String(),
		AttemptCount:      normalizeAttemptCount(item.AttemptCount),
		ConsecutiveErrors: item.ConsecutiveErrors,
		RetryPaused:       item.RetryPaused,
		PauseReason:       item.PauseReason,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		StartedAt:         formatOptionalTime(item.StartedAt),
		CompletedAt:       formatOptionalTime(item.CompletedAt),
	}
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
