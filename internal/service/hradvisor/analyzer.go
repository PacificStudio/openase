package hradvisor

import (
	"fmt"
	"math"
	"sort"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
)

type snapshotStats struct {
	openTickets           int
	codingTickets         int
	failingTickets        int
	blockedTickets        int
	activeAgents          int
	workflowCount         int
	testWorkflowCount     int
	docWorkflowCount      int
	securityWorkflowCount int
	backlogTickets        int
	hasDispatcherWorkflow bool
	statusPressure        []statusPressure
	activeWorkflowTypes   []string
}

type statusPressure struct {
	StatusName          string
	QueuedTickets       int
	PickupWorkflowNames []string
	FinishWorkflowNames []string
}

type laneProfile struct {
	RoleSlug         string
	WorkflowName     string
	WorkflowType     string
	MinQueuedTickets int
	HeadcountDivisor int
}

type projectStatus string

const (
	projectStatusBacklog    projectStatus = "Backlog"
	projectStatusPlanned    projectStatus = "Planned"
	projectStatusInProgress projectStatus = "In Progress"
	projectStatusCompleted  projectStatus = "Completed"
	projectStatusCanceled   projectStatus = "Canceled"
	projectStatusArchived   projectStatus = "Archived"
	projectStatusUnknown    projectStatus = ""
)

// Analyze recommends role coverage from the current project snapshot.
func Analyze(snapshot domain.Snapshot) domain.Analysis {
	stats := collectStats(snapshot)
	status := parseProjectStatus(snapshot.Project.Status)
	activeRoles := make(map[string]struct{}, len(snapshot.ActiveRoleSlugs))
	for _, slug := range snapshot.ActiveRoleSlugs {
		if trimmed := strings.TrimSpace(slug); trimmed != "" {
			activeRoles[trimmed] = struct{}{}
		}
	}

	recommendations := make([]domain.Recommendation, 0, 8)
	recommendedRoles := make(map[string]struct{}, 8)
	add := func(recommendation domain.Recommendation, allowExistingRole bool) {
		if recommendation.RoleSlug == "" {
			return
		}
		if !allowExistingRole {
			if _, exists := activeRoles[recommendation.RoleSlug]; exists {
				return
			}
		}
		if _, exists := recommendedRoles[recommendation.RoleSlug]; exists {
			return
		}
		recommendedRoles[recommendation.RoleSlug] = struct{}{}
		recommendations = append(recommendations, recommendation)
	}

	for _, pressure := range stats.statusPressure {
		recommendation, ok := recommendationForPressure(pressure, stats)
		if !ok {
			continue
		}
		add(recommendation, true)
	}

	if status == projectStatusPlanned && stats.openTickets == 0 && stats.workflowCount == 0 {
		add(domain.Recommendation{
			RoleSlug:              "product-manager",
			Priority:              "high",
			Reason:                "Planned projects with no active ticket lane need a role to turn scope into executable work.",
			Evidence:              []string{"Project status is Planned.", "No workflows are configured yet.", "No open tickets are currently staged."},
			SuggestedHeadcount:    1,
			SuggestedWorkflowName: "Product Manager",
		}, false)
	}

	if status == projectStatusInProgress && stats.activeAgents == 0 {
		add(domain.Recommendation{
			RoleSlug:              "fullstack-developer",
			Priority:              "high",
			Reason:                "The project is In Progress but there is no agent currently able to pick up implementation work.",
			Evidence:              []string{"Project status is In Progress.", fmt.Sprintf("Open tickets: %d.", stats.openTickets), "No active agents are attached."},
			SuggestedHeadcount:    max(1, scaleHeadcount(stats.codingTickets, 4)),
			SuggestedWorkflowName: "Fullstack Developer",
		}, false)
	}

	if stats.codingTickets >= 3 && stats.testWorkflowCount == 0 {
		add(domain.Recommendation{
			RoleSlug:              "qa-engineer",
			Priority:              "high",
			Reason:                "Coding demand is visible, but no test workflow is in place to absorb regression and release checks.",
			Evidence:              []string{fmt.Sprintf("Open coding tickets: %d.", stats.codingTickets), fmt.Sprintf("Active test workflows: %d.", stats.testWorkflowCount)},
			SuggestedHeadcount:    max(1, scaleHeadcount(stats.codingTickets, 6)),
			SuggestedWorkflowName: "QA Engineer",
		}, false)
	}

	if (stats.openTickets >= 5 || stats.workflowCount >= 3) && stats.docWorkflowCount == 0 {
		add(domain.Recommendation{
			RoleSlug:              "technical-writer",
			Priority:              "medium",
			Reason:                "The delivery surface is growing without a documentation lane to keep operator guidance and implementation notes current.",
			Evidence:              []string{fmt.Sprintf("Open tickets: %d.", stats.openTickets), fmt.Sprintf("Configured workflows: %d.", stats.workflowCount), fmt.Sprintf("Active doc workflows: %d.", stats.docWorkflowCount)},
			SuggestedHeadcount:    1,
			SuggestedWorkflowName: "Technical Writer",
		}, false)
	}

	if (stats.openTickets >= 8 || stats.failingTickets >= 2) && stats.securityWorkflowCount == 0 {
		add(domain.Recommendation{
			RoleSlug:              "security-engineer",
			Priority:              "medium",
			Reason:                "Load and failure signals are high enough that a dedicated security pass should be added before the project scales further.",
			Evidence:              []string{fmt.Sprintf("Open tickets: %d.", stats.openTickets), fmt.Sprintf("Failing tickets: %d.", stats.failingTickets), fmt.Sprintf("Active security workflows: %d.", stats.securityWorkflowCount)},
			SuggestedHeadcount:    1,
			SuggestedWorkflowName: "Security Engineer",
		}, false)
	}

	if isResearchProject(snapshot.Project) && stats.workflowCount == 0 {
		add(domain.Recommendation{
			RoleSlug:              "research-ideation",
			Priority:              "medium",
			Reason:                "The project reads like research work, but there is no dedicated role framing questions and experiments yet.",
			Evidence:              []string{"Project title or description contains research-oriented keywords.", "No workflows are configured yet."},
			SuggestedHeadcount:    1,
			SuggestedWorkflowName: "Research Ideation",
		}, false)
	}

	sortRecommendations(recommendations)

	return domain.Analysis{
		Summary: domain.Summary{
			OpenTickets:         stats.openTickets,
			CodingTickets:       stats.codingTickets,
			FailingTickets:      stats.failingTickets,
			BlockedTickets:      stats.blockedTickets,
			ActiveAgents:        stats.activeAgents,
			WorkflowCount:       stats.workflowCount,
			RecentActivityCount: len(snapshot.RecentActivity),
			ActiveWorkflowTypes: stats.activeWorkflowTypes,
		},
		Recommendations: recommendations,
		Staffing:        buildStaffingPlan(status, stats),
	}
}

func collectStats(snapshot domain.Snapshot) snapshotStats {
	stats := snapshotStats{}
	activeWorkflowTypes := make(map[string]struct{})
	queuedTicketsByStatus := make(map[string]int)
	statusDisplayNames := make(map[string]string)
	pickupWorkflowsByStatus := make(map[string]map[string]struct{})
	finishWorkflowsByStatus := make(map[string]map[string]struct{})

	for _, workflow := range snapshot.Workflows {
		stats.workflowCount++
		if !workflow.IsActive {
			continue
		}

		workflowType := strings.TrimSpace(workflow.Type)
		if workflowType != "" {
			activeWorkflowTypes[workflowType] = struct{}{}
		}

		switch workflowType {
		case "test":
			stats.testWorkflowCount++
		case "doc":
			stats.docWorkflowCount++
		case "security":
			stats.securityWorkflowCount++
		}

		if isDispatcherWorkflow(workflow) {
			stats.hasDispatcherWorkflow = true
		}
		registerWorkflowCoverage(pickupWorkflowsByStatus, statusDisplayNames, workflow.PickupStatusNames, workflow.Name)
		registerWorkflowCoverage(finishWorkflowsByStatus, statusDisplayNames, workflow.FinishStatusNames, workflow.Name)
	}

	for _, agent := range snapshot.Agents {
		switch strings.TrimSpace(agent.Status) {
		case "idle", "claimed", "running":
			stats.activeAgents++
		}
	}

	for _, ticket := range snapshot.Tickets {
		if isDoneStatus(ticket.StatusName) {
			continue
		}

		stats.openTickets++
		if ticket.ConsecutiveErrors > 0 {
			stats.failingTickets++
		}
		if ticket.RetryPaused {
			stats.blockedTickets++
		}
		if !ticket.HasActiveRun {
			statusKey := normalizeStatusName(ticket.StatusName)
			if statusKey != "" {
				queuedTicketsByStatus[statusKey]++
				if statusDisplayNames[statusKey] == "" {
					statusDisplayNames[statusKey] = strings.TrimSpace(ticket.StatusName)
				}
				if statusKey == normalizeStatusName(string(projectStatusBacklog)) {
					stats.backlogTickets++
				}
			}
		}

		switch strings.TrimSpace(ticket.WorkflowType) {
		case "coding":
			stats.codingTickets++
		case "test", "doc", "security":
			continue
		default:
			if isCodingTicketType(ticket.Type) {
				stats.codingTickets++
			}
		}
	}

	stats.activeWorkflowTypes = mapKeys(activeWorkflowTypes)
	stats.statusPressure = buildStatusPressure(queuedTicketsByStatus, statusDisplayNames, pickupWorkflowsByStatus, finishWorkflowsByStatus)
	return stats
}

func recommendationForPressure(pressure statusPressure, stats snapshotStats) (domain.Recommendation, bool) {
	if len(pressure.PickupWorkflowNames) > 0 {
		return domain.Recommendation{}, false
	}

	profile, ok := profileForStatus(pressure.StatusName)
	if !ok || pressure.QueuedTickets < profile.MinQueuedTickets {
		return domain.Recommendation{}, false
	}
	if profile.RoleSlug == "dispatcher" && stats.hasDispatcherWorkflow {
		return domain.Recommendation{}, false
	}

	reason := fmt.Sprintf(
		"%s has %d queued tickets, but no active workflow picks up that lane. Add a %s workflow bound to %s.",
		pressure.StatusName,
		pressure.QueuedTickets,
		profile.WorkflowName,
		pressure.StatusName,
	)
	if profile.RoleSlug == "dispatcher" {
		reason = fmt.Sprintf(
			"Backlog has %d queued tickets, but no active Dispatcher workflow is bound to pick up and finish that lane.",
			pressure.QueuedTickets,
		)
	}

	evidence := []string{
		fmt.Sprintf("Queued tickets in status %q: %d.", pressure.StatusName, pressure.QueuedTickets),
		fmt.Sprintf("Active workflows picking up %q: none.", pressure.StatusName),
	}
	if len(pressure.FinishWorkflowNames) > 0 {
		evidence = append(
			evidence,
			fmt.Sprintf("Active workflows finishing into %q: %s.", pressure.StatusName, strings.Join(pressure.FinishWorkflowNames, ", ")),
		)
	}
	if profile.WorkflowType != "" {
		evidence = append(
			evidence,
			fmt.Sprintf("Suggested workflow type for %q: %s.", pressure.StatusName, profile.WorkflowType),
		)
	}
	if profile.RoleSlug == "dispatcher" {
		evidence = append(evidence, "Dispatcher coverage requires a workflow bound to pick up and finish Backlog.")
	}

	return domain.Recommendation{
		RoleSlug:              profile.RoleSlug,
		Priority:              pressurePriority(profile, pressure.QueuedTickets),
		Reason:                reason,
		Evidence:              evidence,
		SuggestedHeadcount:    max(1, scaleHeadcount(pressure.QueuedTickets, profile.HeadcountDivisor)),
		SuggestedWorkflowName: suggestedWorkflowName(profile, pressure.StatusName),
	}, true
}

func parseProjectStatus(raw string) projectStatus {
	switch strings.TrimSpace(raw) {
	case string(projectStatusBacklog):
		return projectStatusBacklog
	case string(projectStatusPlanned):
		return projectStatusPlanned
	case string(projectStatusInProgress):
		return projectStatusInProgress
	case string(projectStatusCompleted):
		return projectStatusCompleted
	case string(projectStatusCanceled):
		return projectStatusCanceled
	case string(projectStatusArchived):
		return projectStatusArchived
	default:
		return projectStatusUnknown
	}
}

func buildStaffingPlan(projectStatus projectStatus, stats snapshotStats) domain.StaffingPlan {
	plan := domain.StaffingPlan{
		Developers: max(0, scaleHeadcount(stats.codingTickets, 4)),
		QA:         0,
		Docs:       0,
		Security:   0,
		Product:    0,
		Research:   0,
	}

	if projectStatus == projectStatusInProgress && plan.Developers == 0 && stats.workflowCount == 0 {
		plan.Developers = 1
	}
	if stats.codingTickets >= 3 {
		plan.QA = max(1, scaleHeadcount(stats.codingTickets, 6))
	}
	if stats.openTickets >= 5 || stats.workflowCount >= 3 {
		plan.Docs = 1
	}
	if stats.openTickets >= 8 || stats.failingTickets >= 2 {
		plan.Security = 1
	}
	if projectStatus == projectStatusPlanned && stats.workflowCount == 0 {
		plan.Product = 1
	}
	if plan.Product == 0 && stats.workflowCount == 0 && stats.openTickets == 0 {
		plan.Research = 0
	}

	return plan
}

func sortRecommendations(items []domain.Recommendation) {
	priorityRank := map[string]int{
		"high":   0,
		"medium": 1,
		"low":    2,
	}

	sort.SliceStable(items, func(i int, j int) bool {
		left := priorityRank[items[i].Priority]
		right := priorityRank[items[j].Priority]
		if left != right {
			return left < right
		}
		if items[i].SuggestedHeadcount != items[j].SuggestedHeadcount {
			return items[i].SuggestedHeadcount > items[j].SuggestedHeadcount
		}
		return items[i].RoleSlug < items[j].RoleSlug
	})
}

func buildStatusPressure(
	queuedTicketsByStatus map[string]int,
	statusDisplayNames map[string]string,
	pickupWorkflowsByStatus map[string]map[string]struct{},
	finishWorkflowsByStatus map[string]map[string]struct{},
) []statusPressure {
	pressures := make([]statusPressure, 0, len(queuedTicketsByStatus))
	for statusKey, queuedTickets := range queuedTicketsByStatus {
		if queuedTickets <= 0 {
			continue
		}

		statusName := statusDisplayNames[statusKey]
		if statusName == "" {
			statusName = statusKey
		}
		pressures = append(pressures, statusPressure{
			StatusName:          statusName,
			QueuedTickets:       queuedTickets,
			PickupWorkflowNames: workflowNamesForStatus(pickupWorkflowsByStatus, statusKey),
			FinishWorkflowNames: workflowNamesForStatus(finishWorkflowsByStatus, statusKey),
		})
	}

	sort.SliceStable(pressures, func(i int, j int) bool {
		if pressures[i].QueuedTickets != pressures[j].QueuedTickets {
			return pressures[i].QueuedTickets > pressures[j].QueuedTickets
		}
		return pressures[i].StatusName < pressures[j].StatusName
	})
	return pressures
}

func mapKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func workflowNamesForStatus(items map[string]map[string]struct{}, statusKey string) []string {
	values := items[statusKey]
	if len(values) == 0 {
		return nil
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func registerWorkflowCoverage(
	items map[string]map[string]struct{},
	statusDisplayNames map[string]string,
	statusNames []string,
	workflowName string,
) {
	trimmedWorkflowName := strings.TrimSpace(workflowName)
	if trimmedWorkflowName == "" {
		return
	}

	for _, statusName := range statusNames {
		statusKey := normalizeStatusName(statusName)
		if statusKey == "" {
			continue
		}

		if statusDisplayNames[statusKey] == "" {
			statusDisplayNames[statusKey] = strings.TrimSpace(statusName)
		}
		if items[statusKey] == nil {
			items[statusKey] = make(map[string]struct{})
		}
		items[statusKey][trimmedWorkflowName] = struct{}{}
	}
}

func profileForStatus(statusName string) (laneProfile, bool) {
	normalized := normalizeStatusName(statusName)
	switch {
	case normalized == normalizeStatusName(string(projectStatusBacklog)):
		return laneProfile{
			RoleSlug:         "dispatcher",
			WorkflowName:     "Dispatcher",
			WorkflowType:     "custom",
			MinQueuedTickets: 11,
			HeadcountDivisor: 10,
		}, true
	case containsStatusKeyword(normalized, "test", "qa", "测试", "验证"):
		return laneProfile{
			RoleSlug:         "qa-engineer",
			WorkflowName:     "QA Engineer",
			WorkflowType:     "test",
			MinQueuedTickets: 2,
			HeadcountDivisor: 6,
		}, true
	case containsStatusKeyword(normalized, "doc", "docs", "write", "writer", "文档", "撰写"):
		return laneProfile{
			RoleSlug:         "technical-writer",
			WorkflowName:     "Technical Writer",
			WorkflowType:     "doc",
			MinQueuedTickets: 2,
			HeadcountDivisor: 8,
		}, true
	case containsStatusKeyword(normalized, "security", "scan", "audit", "安全", "扫描", "审计"):
		return laneProfile{
			RoleSlug:         "security-engineer",
			WorkflowName:     "Security Engineer",
			WorkflowType:     "security",
			MinQueuedTickets: 2,
			HeadcountDivisor: 8,
		}, true
	case containsStatusKeyword(normalized, "todo", "develop", "development", "coding", "implement", "待开发", "开发", "编码", "实现"):
		return laneProfile{
			RoleSlug:         "fullstack-developer",
			WorkflowName:     "Fullstack Developer",
			WorkflowType:     "coding",
			MinQueuedTickets: 2,
			HeadcountDivisor: 4,
		}, true
	default:
		return laneProfile{}, false
	}
}

func suggestedWorkflowName(profile laneProfile, statusName string) string {
	trimmedStatusName := strings.TrimSpace(statusName)
	if trimmedStatusName == "" || profile.RoleSlug == "dispatcher" {
		return profile.WorkflowName
	}
	return fmt.Sprintf("%s - %s", profile.WorkflowName, trimmedStatusName)
}

func pressurePriority(profile laneProfile, queuedTickets int) string {
	if profile.RoleSlug == "dispatcher" || queuedTickets >= 4 {
		return "high"
	}
	return "medium"
}

func containsStatusKeyword(normalized string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(normalized, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return false
}

func normalizeStatusName(statusName string) string {
	return strings.ToLower(strings.TrimSpace(statusName))
}

func isDispatcherWorkflow(workflow domain.WorkflowContext) bool {
	if strings.TrimSpace(workflow.RoleSlug) == "dispatcher" {
		return true
	}
	return hasStatusBinding(workflow.PickupStatusNames, string(projectStatusBacklog)) &&
		hasStatusBinding(workflow.FinishStatusNames, string(projectStatusBacklog))
}

func hasStatusBinding(statusNames []string, want string) bool {
	wantNormalized := normalizeStatusName(want)
	for _, statusName := range statusNames {
		if normalizeStatusName(statusName) == wantNormalized {
			return true
		}
	}
	return false
}

func isDoneStatus(statusName string) bool {
	switch strings.ToLower(strings.TrimSpace(statusName)) {
	case "done", "completed", "closed", "archived":
		return true
	default:
		return false
	}
}

func isCodingTicketType(ticketType string) bool {
	switch strings.ToLower(strings.TrimSpace(ticketType)) {
	case "feature", "bugfix", "refactor", "chore":
		return true
	default:
		return false
	}
}

func isResearchProject(project domain.ProjectContext) bool {
	text := strings.ToLower(project.Name + " " + project.Description)
	for _, keyword := range []string{"research", "experiment", "prototype", "paper"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func scaleHeadcount(workload int, divisor int) int {
	if workload <= 0 || divisor <= 0 {
		return 0
	}
	return int(math.Ceil(float64(workload) / float64(divisor)))
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
