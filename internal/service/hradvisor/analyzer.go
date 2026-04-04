package hradvisor

import (
	"fmt"
	"math"
	"sort"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
)

type snapshotStats struct {
	openTickets            int
	codingTickets          int
	failingTickets         int
	blockedTickets         int
	activeAgents           int
	workflowCount          int
	testWorkflowCount      int
	docWorkflowCount       int
	securityWorkflowCount  int
	backlogTickets         int
	hasDispatcherWorkflow  bool
	documentationDrift     int
	failureBurstCount      int
	trendEvidenceByKind    map[string][]string
	statusPressure         []statusPressure
	activeWorkflowFamilies []string
}

type statusPressure struct {
	StatusName             string
	StatusStage            string
	QueuedTickets          int
	PickupWorkflowNames    []string
	PickupWorkflowFamilies []string
	PickupRoleSlugs        []string
	FinishWorkflowNames    []string
	FinishWorkflowFamilies []string
	FinishRoleSlugs        []string
}

type laneProfile struct {
	RoleSlug          string
	WorkflowName      string
	WorkflowTypeLabel string
	WorkflowFamily    workflowdomain.WorkflowFamily
	MinQueuedTickets  int
	HeadcountDivisor  int
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
	researchProject := isResearchProject(snapshot.Project)
	activeRoles := make(map[string]struct{}, len(snapshot.ActiveRoleSlugs))
	for _, slug := range snapshot.ActiveRoleSlugs {
		if trimmed := strings.TrimSpace(slug); trimmed != "" {
			activeRoles[trimmed] = struct{}{}
		}
	}

	recommendations := make([]domain.Recommendation, 0, 10)
	recommendedKeys := make(map[string]struct{}, 10)
	add := func(key string, recommendation domain.Recommendation, allowExistingRole bool) {
		if recommendation.RoleSlug == "" {
			return
		}
		support, ok := recommendationSupport(recommendation.RoleSlug)
		if !ok || support.Status != recommendationSupportSupportedNow {
			return
		}
		if !allowExistingRole {
			if _, exists := activeRoles[recommendation.RoleSlug]; exists {
				return
			}
		}
		if key == "" {
			key = capabilityRecommendationKey(recommendation.RoleSlug)
		}
		if _, exists := recommendedKeys[key]; exists {
			return
		}
		recommendedKeys[key] = struct{}{}
		recommendations = append(recommendations, recommendation)
	}

	for _, pressure := range stats.statusPressure {
		recommendation, key, ok := recommendationForPressure(pressure, stats, researchProject)
		if ok {
			add(key, recommendation, true)
		}
	}

	if status == projectStatusPlanned && stats.openTickets == 0 && stats.workflowCount == 0 {
		add(capabilityRecommendationKey("product-manager"), domain.Recommendation{
			RoleSlug:                   "product-manager",
			Priority:                   "high",
			Reason:                     "Planned projects with no active ticket lane need a role to turn scope into executable work.",
			Evidence:                   []string{"Project status is Planned.", "No workflows are configured yet.", "No open tickets are currently staged."},
			SuggestedHeadcount:         1,
			SuggestedWorkflowName:      "Product Manager",
			SuggestedWorkflowTypeLabel: "Product Manager",
			SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyPlanning),
		}, false)
	}

	if status == projectStatusInProgress && stats.activeAgents == 0 {
		add(capabilityRecommendationKey("fullstack-developer"), domain.Recommendation{
			RoleSlug:                   "fullstack-developer",
			Priority:                   "high",
			Reason:                     "The project is In Progress but there is no agent currently able to pick up implementation work.",
			Evidence:                   []string{"Project status is In Progress.", fmt.Sprintf("Open tickets: %d.", stats.openTickets), "No active agents are attached."},
			SuggestedHeadcount:         max(1, scaleHeadcount(stats.codingTickets, 4)),
			SuggestedWorkflowName:      "Fullstack Developer",
			SuggestedWorkflowTypeLabel: "Fullstack Developer",
			SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyCoding),
		}, false)
	}

	if stats.codingTickets >= 3 && stats.testWorkflowCount == 0 {
		add(capabilityRecommendationKey("qa-engineer"), domain.Recommendation{
			RoleSlug:                   "qa-engineer",
			Priority:                   "high",
			Reason:                     "Coding demand is visible, but no test workflow is in place to absorb regression and release checks.",
			Evidence:                   []string{fmt.Sprintf("Open coding tickets: %d.", stats.codingTickets), fmt.Sprintf("Active test workflows: %d.", stats.testWorkflowCount)},
			SuggestedHeadcount:         max(1, scaleHeadcount(stats.codingTickets, 6)),
			SuggestedWorkflowName:      "QA Engineer",
			SuggestedWorkflowTypeLabel: "QA Engineer",
			SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyTest),
		}, false)
	}

	if recommendation, ok := documentationRecommendation(stats); ok {
		add(capabilityRecommendationKey(recommendation.RoleSlug), recommendation, false)
	}

	if recommendation, ok := securityRecommendation(stats); ok {
		add(capabilityRecommendationKey(recommendation.RoleSlug), recommendation, false)
	}

	if recommendation, ok := envProvisionerRecommendation(stats); ok {
		add(capabilityRecommendationKey(recommendation.RoleSlug), recommendation, false)
	}

	if recommendation, ok := harnessOptimizerRecommendation(stats); ok {
		add(capabilityRecommendationKey(recommendation.RoleSlug), recommendation, false)
	}

	if researchProject && stats.workflowCount == 0 {
		add(capabilityRecommendationKey("research-ideation"), domain.Recommendation{
			RoleSlug:                   "research-ideation",
			Priority:                   "medium",
			Reason:                     "The project reads like research work, but there is no dedicated role framing questions and experiments yet.",
			Evidence:                   []string{"Project title or description contains research-oriented keywords.", "No workflows are configured yet."},
			SuggestedHeadcount:         1,
			SuggestedWorkflowName:      "Research Ideation",
			SuggestedWorkflowTypeLabel: "Research Ideation",
			SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyResearch),
		}, false)
	}

	sortRecommendations(recommendations)

	return domain.Analysis{
		Summary: domain.Summary{
			OpenTickets:            stats.openTickets,
			CodingTickets:          stats.codingTickets,
			FailingTickets:         stats.failingTickets,
			BlockedTickets:         stats.blockedTickets,
			ActiveAgents:           stats.activeAgents,
			WorkflowCount:          stats.workflowCount,
			RecentActivityCount:    snapshot.RecentActivityCount,
			ActiveWorkflowFamilies: stats.activeWorkflowFamilies,
		},
		Recommendations: recommendations,
		Staffing:        buildStaffingPlan(status, stats),
	}
}

func collectStats(snapshot domain.Snapshot) snapshotStats {
	stats := snapshotStats{trendEvidenceByKind: make(map[string][]string)}
	activeWorkflowFamilies := make(map[string]struct{})
	queuedTicketsByStatus := make(map[string]int)
	statusDisplayNames := make(map[string]string)
	statusStagesByStatus := make(map[string]string)
	pickupWorkflowsByStatus := make(map[string]map[string]struct{})
	pickupWorkflowFamiliesByStatus := make(map[string]map[string]struct{})
	pickupRoleSlugsByStatus := make(map[string]map[string]struct{})
	finishWorkflowsByStatus := make(map[string]map[string]struct{})
	finishWorkflowFamiliesByStatus := make(map[string]map[string]struct{})
	finishRoleSlugsByStatus := make(map[string]map[string]struct{})

	for _, workflow := range snapshot.Workflows {
		stats.workflowCount++
		if !workflow.IsActive {
			continue
		}

		classification := classifyWorkflow(workflow)
		if classification.Family != workflowdomain.WorkflowFamilyUnknown {
			activeWorkflowFamilies[string(classification.Family)] = struct{}{}
		}

		switch classification.Family {
		case workflowdomain.WorkflowFamilyTest:
			stats.testWorkflowCount++
		case workflowdomain.WorkflowFamilyDocs:
			stats.docWorkflowCount++
		case workflowdomain.WorkflowFamilySecurity:
			stats.securityWorkflowCount++
		}

		if classification.Family == workflowdomain.WorkflowFamilyDispatcher || isDispatcherWorkflow(workflow) {
			stats.hasDispatcherWorkflow = true
		}
		registerWorkflowCoverage(
			pickupWorkflowsByStatus,
			pickupWorkflowFamiliesByStatus,
			pickupRoleSlugsByStatus,
			statusDisplayNames,
			statusStagesByStatus,
			workflow.PickupStatuses,
			workflow.Name,
			classification.Family,
			workflow.RoleSlug,
		)
		registerWorkflowCoverage(
			finishWorkflowsByStatus,
			finishWorkflowFamiliesByStatus,
			finishRoleSlugsByStatus,
			statusDisplayNames,
			statusStagesByStatus,
			workflow.FinishStatuses,
			workflow.Name,
			classification.Family,
			workflow.RoleSlug,
		)
	}

	for _, agent := range snapshot.Agents {
		switch strings.TrimSpace(agent.Status) {
		case "idle", "claimed", "running":
			stats.activeAgents++
		}
	}

	for _, ticket := range snapshot.Tickets {
		if isDoneStatus(ticket.StatusName, ticket.StatusStage) {
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
				if stage := parseStatusStage(ticket.StatusName, ticket.StatusStage); stage != "" && statusStagesByStatus[statusKey] == "" {
					statusStagesByStatus[statusKey] = stage.String()
				}
				if parseStatusStage(ticket.StatusName, ticket.StatusStage) == ticketing.StatusStageBacklog ||
					statusKey == normalizeStatusName(string(projectStatusBacklog)) {
					stats.backlogTickets++
				}
			}
		}

		workflowTypeLabel := ticket.WorkflowTypeLabel
		if strings.TrimSpace(workflowTypeLabel) == "" {
			workflowTypeLabel = ticket.WorkflowType
		}
		switch classifyWorkflowTypeLabel(workflowTypeLabel) {
		case workflowdomain.WorkflowFamilyCoding:
			stats.codingTickets++
		case workflowdomain.WorkflowFamilyTest,
			workflowdomain.WorkflowFamilyDocs,
			workflowdomain.WorkflowFamilySecurity:
			continue
		default:
			if isCodingTicketType(ticket.Type) {
				stats.codingTickets++
			}
		}
	}

	for _, trend := range snapshot.RecentTrends {
		switch trend.Kind {
		case domain.ActivityTrendDocumentationDrift:
			stats.documentationDrift += trend.Count
		case domain.ActivityTrendFailureBurst:
			stats.failureBurstCount += trend.Count
		}
		if len(trend.Evidence) > 0 {
			stats.trendEvidenceByKind[trend.Kind] = append(stats.trendEvidenceByKind[trend.Kind], trend.Evidence...)
		}
	}

	stats.activeWorkflowFamilies = mapKeys(activeWorkflowFamilies)
	stats.statusPressure = buildStatusPressure(
		queuedTicketsByStatus,
		statusDisplayNames,
		statusStagesByStatus,
		pickupWorkflowsByStatus,
		pickupWorkflowFamiliesByStatus,
		pickupRoleSlugsByStatus,
		finishWorkflowsByStatus,
		finishWorkflowFamiliesByStatus,
		finishRoleSlugsByStatus,
	)
	return stats
}

func documentationRecommendation(stats snapshotStats) (domain.Recommendation, bool) {
	if stats.docWorkflowCount > 0 {
		return domain.Recommendation{}, false
	}

	hasStaticPressure := stats.openTickets >= 5 || stats.workflowCount >= 3
	hasTrendPressure := stats.documentationDrift > 0
	if !hasStaticPressure && !hasTrendPressure {
		return domain.Recommendation{}, false
	}

	reason := "The delivery surface is growing without a documentation lane to keep operator guidance and implementation notes current."
	if hasTrendPressure {
		reason = "Recent delivery activity shows merges outpacing documentation updates, and there is no documentation lane to absorb that drift."
	}

	evidence := make([]string, 0, 5)
	if hasStaticPressure {
		evidence = append(evidence,
			fmt.Sprintf("Open tickets: %d.", stats.openTickets),
			fmt.Sprintf("Configured workflows: %d.", stats.workflowCount),
		)
	}
	if hasTrendPressure {
		evidence = append(evidence, stats.trendEvidenceByKind[domain.ActivityTrendDocumentationDrift]...)
		evidence = append(evidence, fmt.Sprintf("Documentation drift signals: %d.", stats.documentationDrift))
	}
	evidence = append(evidence, fmt.Sprintf("Active doc workflows: %d.", stats.docWorkflowCount))

	priority := "medium"
	if stats.documentationDrift >= 3 {
		priority = "high"
	}

	return domain.Recommendation{
		RoleSlug:                   "technical-writer",
		Priority:                   priority,
		Reason:                     reason,
		Evidence:                   evidence,
		SuggestedHeadcount:         1,
		SuggestedWorkflowName:      "Technical Writer",
		SuggestedWorkflowTypeLabel: "Technical Writer",
		SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyDocs),
	}, true
}

func securityRecommendation(stats snapshotStats) (domain.Recommendation, bool) {
	if stats.securityWorkflowCount > 0 {
		return domain.Recommendation{}, false
	}
	if stats.openTickets < 8 && stats.failingTickets < 2 && stats.failureBurstCount == 0 {
		return domain.Recommendation{}, false
	}

	evidence := []string{
		fmt.Sprintf("Open tickets: %d.", stats.openTickets),
		fmt.Sprintf("Failing tickets: %d.", stats.failingTickets),
		fmt.Sprintf("Active security workflows: %d.", stats.securityWorkflowCount),
	}
	if stats.failureBurstCount > 0 {
		evidence = append(evidence, stats.trendEvidenceByKind[domain.ActivityTrendFailureBurst]...)
	}

	return domain.Recommendation{
		RoleSlug:                   "security-engineer",
		Priority:                   "medium",
		Reason:                     "Load and failure signals are high enough that a dedicated security pass should be added before the project scales further.",
		Evidence:                   evidence,
		SuggestedHeadcount:         1,
		SuggestedWorkflowName:      "Security Engineer",
		SuggestedWorkflowTypeLabel: "Security Engineer",
		SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilySecurity),
	}, true
}

func envProvisionerRecommendation(stats snapshotStats) (domain.Recommendation, bool) {
	if stats.blockedTickets == 0 {
		return domain.Recommendation{}, false
	}
	if stats.failingTickets == 0 && stats.failureBurstCount == 0 {
		return domain.Recommendation{}, false
	}

	evidence := []string{
		fmt.Sprintf("Blocked tickets with paused retries: %d.", stats.blockedTickets),
		fmt.Sprintf("Failing tickets: %d.", stats.failingTickets),
	}
	if stats.failureBurstCount > 0 {
		evidence = append(evidence, fmt.Sprintf("Recent failure bursts: %d.", stats.failureBurstCount))
		evidence = append(evidence, stats.trendEvidenceByKind[domain.ActivityTrendFailureBurst]...)
	}

	priority := "medium"
	if stats.blockedTickets >= 2 || stats.failureBurstCount > 0 {
		priority = "high"
	}

	return domain.Recommendation{
		RoleSlug:                   "env-provisioner",
		Priority:                   priority,
		Reason:                     "Repeated stalled retries suggest the project needs an explicit environment-repair lane before more execution work can progress safely.",
		Evidence:                   evidence,
		SuggestedHeadcount:         1,
		SuggestedWorkflowName:      "Environment Provisioner",
		SuggestedWorkflowTypeLabel: "Environment Provisioner",
		SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyEnvironment),
	}, true
}

func harnessOptimizerRecommendation(stats snapshotStats) (domain.Recommendation, bool) {
	if stats.workflowCount == 0 {
		return domain.Recommendation{}, false
	}
	if stats.blockedTickets == 0 && stats.failureBurstCount == 0 && stats.failingTickets < 3 {
		return domain.Recommendation{}, false
	}

	evidence := []string{
		fmt.Sprintf("Configured workflows: %d.", stats.workflowCount),
		fmt.Sprintf("Blocked tickets with paused retries: %d.", stats.blockedTickets),
		fmt.Sprintf("Failing tickets: %d.", stats.failingTickets),
	}
	if stats.failureBurstCount > 0 {
		evidence = append(evidence, fmt.Sprintf("Recent failure bursts: %d.", stats.failureBurstCount))
		evidence = append(evidence, stats.trendEvidenceByKind[domain.ActivityTrendFailureBurst]...)
	}

	priority := "medium"
	if stats.blockedTickets >= 2 || stats.failureBurstCount > 0 {
		priority = "high"
	}

	return domain.Recommendation{
		RoleSlug:                   "harness-optimizer",
		Priority:                   priority,
		Reason:                     "Workflow retries and stalled execution suggest harness quality drift that should be corrected before scaling the current workflow set further.",
		Evidence:                   evidence,
		SuggestedHeadcount:         1,
		SuggestedWorkflowName:      "Harness Optimizer",
		SuggestedWorkflowTypeLabel: "Harness Optimizer",
		SuggestedWorkflowFamily:    string(workflowdomain.WorkflowFamilyHarness),
	}, true
}

func recommendationForPressure(pressure statusPressure, stats snapshotStats, researchProject bool) (domain.Recommendation, string, bool) {
	if len(pressure.PickupWorkflowNames) > 0 {
		return domain.Recommendation{}, "", false
	}

	profile, ok := profileForStatus(pressure, researchProject)
	if !ok || pressure.QueuedTickets < profile.MinQueuedTickets {
		return domain.Recommendation{}, "", false
	}
	if profile.RoleSlug == "dispatcher" && stats.hasDispatcherWorkflow {
		return domain.Recommendation{}, "", false
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
			"Backlog has %d queued tickets, but no active Dispatcher workflow is bound to pick up that lane and route tickets into downstream work statuses.",
			pressure.QueuedTickets,
		)
	}

	evidence := []string{
		fmt.Sprintf("Queued tickets in status %q: %d.", pressure.StatusName, pressure.QueuedTickets),
		fmt.Sprintf("Status stage for %q: %s.", pressure.StatusName, statusStageLabel(pressure.StatusStage)),
		fmt.Sprintf("Active workflows picking up %q: none.", pressure.StatusName),
	}
	if len(pressure.FinishWorkflowNames) > 0 {
		evidence = append(
			evidence,
			fmt.Sprintf("Active workflows finishing into %q: %s.", pressure.StatusName, strings.Join(pressure.FinishWorkflowNames, ", ")),
		)
	}
	if len(pressure.FinishWorkflowFamilies) > 0 {
		evidence = append(
			evidence,
			fmt.Sprintf("Upstream workflow families finishing into %q: %s.", pressure.StatusName, strings.Join(pressure.FinishWorkflowFamilies, ", ")),
		)
	}
	if profile.WorkflowTypeLabel != "" {
		evidence = append(
			evidence,
			fmt.Sprintf("Suggested workflow type label for %q: %s.", pressure.StatusName, profile.WorkflowTypeLabel),
			fmt.Sprintf("Suggested workflow family for %q: %s.", pressure.StatusName, profile.WorkflowFamily),
		)
	}
	if profile.RoleSlug == "dispatcher" {
		evidence = append(evidence, "Dispatcher coverage requires a workflow bound to pick up Backlog and finish into downstream non-backlog work statuses.")
	}

	return domain.Recommendation{
		RoleSlug:                   profile.RoleSlug,
		Priority:                   pressurePriority(profile, pressure.QueuedTickets),
		Reason:                     reason,
		Evidence:                   evidence,
		SuggestedHeadcount:         max(1, scaleHeadcount(pressure.QueuedTickets, profile.HeadcountDivisor)),
		SuggestedWorkflowName:      suggestedWorkflowName(profile, pressure.StatusName),
		SuggestedWorkflowTypeLabel: profile.WorkflowTypeLabel,
		SuggestedWorkflowFamily:    string(profile.WorkflowFamily),
	}, laneRecommendationKey(profile.RoleSlug, pressure.StatusName), true
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
	if stats.openTickets >= 5 || stats.workflowCount >= 3 || stats.documentationDrift > 0 {
		plan.Docs = 1
	}
	if stats.openTickets >= 8 || stats.failingTickets >= 2 || stats.failureBurstCount > 0 {
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
		if items[i].RoleSlug == items[j].RoleSlug && items[i].SuggestedWorkflowName != items[j].SuggestedWorkflowName {
			return items[i].SuggestedWorkflowName < items[j].SuggestedWorkflowName
		}
		return items[i].RoleSlug < items[j].RoleSlug
	})
}

func buildStatusPressure(
	queuedTicketsByStatus map[string]int,
	statusDisplayNames map[string]string,
	statusStagesByStatus map[string]string,
	pickupWorkflowsByStatus map[string]map[string]struct{},
	pickupWorkflowFamiliesByStatus map[string]map[string]struct{},
	pickupRoleSlugsByStatus map[string]map[string]struct{},
	finishWorkflowsByStatus map[string]map[string]struct{},
	finishWorkflowFamiliesByStatus map[string]map[string]struct{},
	finishRoleSlugsByStatus map[string]map[string]struct{},
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
			StatusName:             statusName,
			StatusStage:            statusStagesByStatus[statusKey],
			QueuedTickets:          queuedTickets,
			PickupWorkflowNames:    workflowNamesForStatus(pickupWorkflowsByStatus, statusKey),
			PickupWorkflowFamilies: workflowNamesForStatus(pickupWorkflowFamiliesByStatus, statusKey),
			PickupRoleSlugs:        workflowNamesForStatus(pickupRoleSlugsByStatus, statusKey),
			FinishWorkflowNames:    workflowNamesForStatus(finishWorkflowsByStatus, statusKey),
			FinishWorkflowFamilies: workflowNamesForStatus(finishWorkflowFamiliesByStatus, statusKey),
			FinishRoleSlugs:        workflowNamesForStatus(finishRoleSlugsByStatus, statusKey),
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

func classifyWorkflow(workflow domain.WorkflowContext) workflowdomain.WorkflowClassification {
	rawTypeLabel := workflow.TypeLabel
	if strings.TrimSpace(rawTypeLabel) == "" {
		rawTypeLabel = workflow.Type
	}
	typeLabel, err := workflowdomain.ParseTypeLabel(rawTypeLabel)
	if err != nil {
		typeLabel = workflowdomain.MustParseTypeLabel("unknown")
	}
	pickupStatusNames := make([]string, 0, len(workflow.PickupStatuses))
	for _, status := range workflow.PickupStatuses {
		pickupStatusNames = append(pickupStatusNames, status.Name)
	}
	finishStatusNames := make([]string, 0, len(workflow.FinishStatuses))
	for _, status := range workflow.FinishStatuses {
		finishStatusNames = append(finishStatusNames, status.Name)
	}
	return workflowdomain.ClassifyWorkflow(workflowdomain.WorkflowClassificationInput{
		RoleSlug:          workflow.RoleSlug,
		TypeLabel:         typeLabel,
		WorkflowName:      workflow.Name,
		PickupStatusNames: pickupStatusNames,
		FinishStatusNames: finishStatusNames,
		HarnessPath:       workflow.HarnessPath,
		HarnessContent:    workflow.HarnessContent,
	})
}

func classifyWorkflowTypeLabel(raw string) workflowdomain.WorkflowFamily {
	typeLabel, err := workflowdomain.ParseTypeLabel(raw)
	if err != nil {
		return workflowdomain.WorkflowFamilyUnknown
	}
	return workflowdomain.ClassifyWorkflow(workflowdomain.WorkflowClassificationInput{
		TypeLabel: typeLabel,
	}).Family
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
	names map[string]map[string]struct{},
	families map[string]map[string]struct{},
	roles map[string]map[string]struct{},
	statusDisplayNames map[string]string,
	statusStages map[string]string,
	statusBindings []domain.StatusBindingContext,
	workflowName string,
	workflowFamily workflowdomain.WorkflowFamily,
	roleSlug string,
) {
	trimmedWorkflowName := strings.TrimSpace(workflowName)
	if trimmedWorkflowName == "" {
		return
	}

	for _, statusBinding := range statusBindings {
		statusKey := normalizeStatusName(statusBinding.Name)
		if statusKey == "" {
			continue
		}

		if statusDisplayNames[statusKey] == "" {
			statusDisplayNames[statusKey] = strings.TrimSpace(statusBinding.Name)
		}
		if stage := parseStatusStage(statusBinding.Name, statusBinding.Stage); stage != "" && statusStages[statusKey] == "" {
			statusStages[statusKey] = stage.String()
		}
		if names[statusKey] == nil {
			names[statusKey] = make(map[string]struct{})
		}
		names[statusKey][trimmedWorkflowName] = struct{}{}
		addStatusValue(families, statusKey, strings.TrimSpace(string(workflowFamily)))
		addStatusValue(roles, statusKey, strings.TrimSpace(roleSlug))
	}
}

func addStatusValue(items map[string]map[string]struct{}, statusKey string, value string) {
	if value == "" {
		return
	}
	if items[statusKey] == nil {
		items[statusKey] = make(map[string]struct{})
	}
	items[statusKey][value] = struct{}{}
}

func profileForStatus(pressure statusPressure, researchProject bool) (laneProfile, bool) {
	normalized := normalizeStatusName(pressure.StatusName)
	stage := parseStatusStage(pressure.StatusName, pressure.StatusStage)

	switch {
	case stage == ticketing.StatusStageBacklog || normalized == normalizeStatusName(string(projectStatusBacklog)):
		return laneProfile{
			RoleSlug:          "dispatcher",
			WorkflowName:      "Dispatcher",
			WorkflowTypeLabel: "Dispatcher",
			WorkflowFamily:    workflowdomain.WorkflowFamilyDispatcher,
			MinQueuedTickets:  11,
			HeadcountDivisor:  10,
		}, true
	case containsStatusKeyword(normalized, "review", "reviewer", "approve", "approval", "pr", "审查", "评审", "审核"):
		return laneProfile{
			RoleSlug:          "code-reviewer",
			WorkflowName:      "Code Reviewer",
			WorkflowTypeLabel: "Code Reviewer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyReview,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "test", "qa", "测试", "验证"):
		return laneProfile{
			RoleSlug:          "qa-engineer",
			WorkflowName:      "QA Engineer",
			WorkflowTypeLabel: "QA Engineer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyTest,
			MinQueuedTickets:  2,
			HeadcountDivisor:  6,
		}, true
	case researchProject && containsStatusKeyword(normalized, "report", "paper", "writing", "writeup", "写作", "报告", "论文"):
		return laneProfile{
			RoleSlug:          "report-writer",
			WorkflowName:      "Report Writer",
			WorkflowTypeLabel: "Report Writer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyReporting,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "doc", "docs", "write", "writer", "文档", "撰写"):
		return laneProfile{
			RoleSlug:          "technical-writer",
			WorkflowName:      "Technical Writer",
			WorkflowTypeLabel: "Technical Writer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyDocs,
			MinQueuedTickets:  2,
			HeadcountDivisor:  8,
		}, true
	case containsStatusKeyword(normalized, "deploy", "release", "rollout", "ship", "上线", "部署", "发布"):
		return laneProfile{
			RoleSlug:          "devops-engineer",
			WorkflowName:      "DevOps Engineer",
			WorkflowTypeLabel: "Release Captain",
			WorkflowFamily:    workflowdomain.WorkflowFamilyDeploy,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "environment", "env", "bootstrap", "machine", "setup", "provision", "repair", "环境", "修复", "配置"):
		return laneProfile{
			RoleSlug:          "env-provisioner",
			WorkflowName:      "Environment Provisioner",
			WorkflowTypeLabel: "Environment Provisioner",
			WorkflowFamily:    workflowdomain.WorkflowFamilyEnvironment,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "harness", "prompt", "workflow tune", "workflow-tune", "优化", "调优"):
		return laneProfile{
			RoleSlug:          "harness-optimizer",
			WorkflowName:      "Harness Optimizer",
			WorkflowTypeLabel: "Harness Optimizer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyHarness,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "security", "scan", "audit", "安全", "扫描", "审计"):
		return laneProfile{
			RoleSlug:          "security-engineer",
			WorkflowName:      "Security Engineer",
			WorkflowTypeLabel: "Security Engineer",
			WorkflowFamily:    workflowdomain.WorkflowFamilySecurity,
			MinQueuedTickets:  2,
			HeadcountDivisor:  8,
		}, true
	case containsStatusKeyword(normalized, "frontend", "front-end", "ui", "ux", "web", "页面", "前端", "界面"):
		return laneProfile{
			RoleSlug:          "frontend-engineer",
			WorkflowName:      "Frontend Engineer",
			WorkflowTypeLabel: "Frontend Engineer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyCoding,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "backend", "back-end", "api", "service", "server", "后端", "接口", "服务"):
		return laneProfile{
			RoleSlug:          "backend-engineer",
			WorkflowName:      "Backend Engineer",
			WorkflowTypeLabel: "Backend Engineer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyCoding,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case researchProject && containsStatusKeyword(normalized, "experiment", "trial", "benchmark", "实验", "试验"):
		return laneProfile{
			RoleSlug:          "experiment-runner",
			WorkflowName:      "Experiment Runner",
			WorkflowTypeLabel: "Experiment Runner",
			WorkflowFamily:    workflowdomain.WorkflowFamilyResearch,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case researchProject && containsStatusKeyword(normalized, "research", "ideation", "investigate", "literature", "study", "调研", "研究"):
		return laneProfile{
			RoleSlug:          "research-ideation",
			WorkflowName:      "Research Ideation",
			WorkflowTypeLabel: "Research Ideation",
			WorkflowFamily:    workflowdomain.WorkflowFamilyResearch,
			MinQueuedTickets:  1,
			HeadcountDivisor:  4,
		}, true
	case containsStatusKeyword(normalized, "todo", "develop", "development", "coding", "implement", "待开发", "开发", "编码", "实现"):
		return laneProfile{
			RoleSlug:          "fullstack-developer",
			WorkflowName:      "Fullstack Developer",
			WorkflowTypeLabel: "Fullstack Developer",
			WorkflowFamily:    workflowdomain.WorkflowFamilyCoding,
			MinQueuedTickets:  2,
			HeadcountDivisor:  4,
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
	if profile.RoleSlug == "dispatcher" || profile.RoleSlug == "env-provisioner" || profile.RoleSlug == "harness-optimizer" || queuedTickets >= 4 {
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

func laneRecommendationKey(roleSlug string, statusName string) string {
	return "lane:" + roleSlug + ":" + normalizeStatusName(statusName)
}

func capabilityRecommendationKey(roleSlug string) string {
	return "capability:" + roleSlug
}

func statusStageLabel(rawStage string) string {
	if strings.TrimSpace(rawStage) == "" {
		return "unknown"
	}
	return rawStage
}

func isDispatcherWorkflow(workflow domain.WorkflowContext) bool {
	if strings.TrimSpace(workflow.RoleSlug) == "dispatcher" {
		return true
	}
	return hasStatusBinding(workflow.PickupStatuses, string(projectStatusBacklog)) &&
		hasStatusBinding(workflow.FinishStatuses, string(projectStatusBacklog))
}

func hasStatusBinding(statusBindings []domain.StatusBindingContext, want string) bool {
	wantNormalized := normalizeStatusName(want)
	for _, statusBinding := range statusBindings {
		if normalizeStatusName(statusBinding.Name) == wantNormalized {
			return true
		}
	}
	return false
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
