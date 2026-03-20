import type {
  HRAdvisorRecommendation,
  HRAdvisorStaffing,
  OnboardingMilestone,
  OnboardingSnapshot,
  OnboardingSummary,
  Organization,
  OrganizationForm,
  Project,
  ProjectStatus,
  ProjectForm,
  Skill,
  Ticket,
  TicketPriority,
  TicketStatus,
  Workflow,
  WorkflowType,
  WorkflowForm,
} from './types'

export function orderTicketStatuses(statuses: TicketStatus[]) {
  return [...statuses].sort((left, right) => {
    const positionDelta = left.position - right.position
    if (positionDelta !== 0) {
      return positionDelta
    }

    return left.name.localeCompare(right.name)
  })
}

export function orderTickets(items: Ticket[]) {
  return [...items].sort((left, right) => {
    const priorityDelta = ticketPriorityRank(left.priority) - ticketPriorityRank(right.priority)
    if (priorityDelta !== 0) {
      return priorityDelta
    }

    const createdDelta = Date.parse(left.created_at) - Date.parse(right.created_at)
    if (!Number.isNaN(createdDelta) && createdDelta !== 0) {
      return createdDelta
    }

    return left.identifier.localeCompare(right.identifier)
  })
}

export function ticketPriorityRank(priority: TicketPriority) {
  switch (priority) {
    case 'urgent':
      return 0
    case 'high':
      return 1
    case 'medium':
      return 2
    default:
      return 3
  }
}

export function workflowHasSkill(skill: Skill, workflowID?: string | null) {
  if (!workflowID) {
    return false
  }

  return skill.bound_workflows.some((workflow) => workflow.id === workflowID)
}

export function staffingEntries(staffing: HRAdvisorStaffing) {
  return [
    { label: 'Developers', value: staffing.developers },
    { label: 'QA', value: staffing.qa },
    { label: 'Docs', value: staffing.docs },
    { label: 'Security', value: staffing.security },
    { label: 'Product', value: staffing.product },
    { label: 'Research', value: staffing.research },
  ].filter((item) => item.value > 0)
}

export function hrAdvisorPriorityBadgeClass(priority: HRAdvisorRecommendation['priority']) {
  switch (priority) {
    case 'high':
      return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
    case 'medium':
      return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
    default:
      return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
  }
}

export function hrAdvisorPriorityCardClass(priority: HRAdvisorRecommendation['priority']) {
  switch (priority) {
    case 'high':
      return 'border-rose-500/25 bg-rose-500/[0.08]'
    case 'medium':
      return 'border-amber-500/25 bg-amber-500/[0.08]'
    default:
      return 'border-sky-500/25 bg-sky-500/[0.08]'
  }
}

export function ticketPriorityBadgeClass(priority: TicketPriority) {
  switch (priority) {
    case 'urgent':
      return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
    case 'high':
      return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
    case 'medium':
      return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
    default:
      return 'border-border/80 bg-background text-muted-foreground'
  }
}

export function toOrganizationForm(item: Organization): OrganizationForm {
  return {
    name: item.name,
    slug: item.slug,
  }
}

export function toProjectForm(item: Project): ProjectForm {
  return {
    name: item.name,
    slug: item.slug,
    description: item.description,
    status: item.status as ProjectStatus,
    maxConcurrentAgents: item.max_concurrent_agents,
  }
}

export function toWorkflowForm(item: Workflow): WorkflowForm {
  return {
    name: item.name,
    type: item.type as WorkflowType,
    pickupStatusId: item.pickup_status_id,
    finishStatusId: item.finish_status_id ?? '',
    maxConcurrent: item.max_concurrent,
    maxRetryAttempts: item.max_retry_attempts,
    timeoutMinutes: item.timeout_minutes,
    stallTimeoutMinutes: item.stall_timeout_minutes,
    isActive: item.is_active,
  }
}

export function defaultProjectForm(): ProjectForm {
  return {
    name: '',
    slug: '',
    description: '',
    status: 'planning',
    maxConcurrentAgents: 5,
  }
}

export function defaultWorkflowForm(statuses: TicketStatus[] = []): WorkflowForm {
  const pickup = statuses.find((status) => status.is_default) ?? statuses[0]
  const finish =
    statuses.find((status) => status.name.toLowerCase() === 'done') ??
    statuses.find((status) => status.name.toLowerCase() === 'completed') ??
    statuses[statuses.length - 1]

  return {
    name: '',
    type: 'coding',
    pickupStatusId: pickup?.id ?? '',
    finishStatusId: finish?.id ?? '',
    maxConcurrent: 3,
    maxRetryAttempts: 3,
    timeoutMinutes: 60,
    stallTimeoutMinutes: 5,
    isActive: true,
  }
}

export function slugify(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export function buildOnboardingSummary(snapshot: OnboardingSnapshot): OnboardingSummary {
  const hasOrganization = snapshot.organizationCount > 0
  const hasProject =
    hasOrganization && snapshot.projectCount > 0 && snapshot.selectedProjectName.length > 0
  const hasWorkflowLane = hasProject && snapshot.statusCount > 0 && snapshot.workflowCount > 0
  const hasTicket = hasWorkflowLane && snapshot.ticketCount > 0
  const hasAgent = hasTicket && snapshot.agentCount > 0
  const hasSignal = hasAgent && snapshot.hasAutomationSignal

  const milestones: OnboardingMilestone[] = [
    {
      key: 'organization',
      title: 'Workspace boundary',
      description: hasOrganization
        ? `Organization scope is live${snapshot.selectedOrgName ? ` in ${snapshot.selectedOrgName}` : ''}.`
        : 'Create the first organization so OpenASE has a top-level workspace boundary.',
      action: 'Create an organization to unlock project and workflow surfaces.',
      completed: hasOrganization,
      isCurrent: false,
    },
    {
      key: 'project',
      title: 'Project selected',
      description: hasProject
        ? `${snapshot.selectedProjectName} is selected and its board context is loaded.`
        : 'Create a project inside the selected organization to load statuses, board, and workflows.',
      action: 'Create a project for the current organization.',
      completed: hasProject,
      isCurrent: false,
    },
    {
      key: 'workflow-lane',
      title: 'Runnable lane',
      description: hasWorkflowLane
        ? `${snapshot.statusCount} status columns and ${snapshot.workflowCount} workflow${snapshot.workflowCount === 1 ? '' : 's'} are ready.`
        : 'Add at least one workflow on top of the project board so tickets have an executable lane.',
      action: 'Create a workflow and keep the board statuses wired.',
      completed: hasWorkflowLane,
      isCurrent: false,
    },
    {
      key: 'ticket',
      title: 'First ticket seeded',
      description: hasTicket
        ? `${snapshot.ticketCount} ticket${snapshot.ticketCount === 1 ? '' : 's'} are ready for routing on the board.`
        : 'Seed the first ticket so the board can demonstrate ticket-driven routing.',
      action: 'Seed a first ticket for the selected project.',
      completed: hasTicket,
      isCurrent: false,
    },
    {
      key: 'agent',
      title: 'Telemetry unlocked',
      description: hasAgent
        ? `${snapshot.agentCount} agent${snapshot.agentCount === 1 ? '' : 's'} attached, ${snapshot.runningAgentCount} currently running.`
        : 'Attach an agent to this project to unlock live heartbeat, activity, and execution telemetry.',
      action: 'Start or register an agent for the selected project.',
      completed: hasAgent,
      isCurrent: false,
    },
    {
      key: 'automation-signal',
      title: 'First automation signal',
      description: hasSignal
        ? `Live work has been observed through tickets, agents, or activity events (${snapshot.activityCount} activity event${snapshot.activityCount === 1 ? '' : 's'} in view).`
        : 'Kick off a first run and watch the board, agent console, and activity stream for movement.',
      action: 'Trigger the first ticket run and confirm the banner flips to fully unlocked.',
      completed: hasSignal,
      isCurrent: false,
    },
  ]

  const currentIndex = milestones.findIndex((item) => !item.completed)
  if (currentIndex >= 0) {
    milestones[currentIndex] = { ...milestones[currentIndex], isCurrent: true }
  }

  const completedCount = milestones.filter((item) => item.completed).length
  const totalCount = milestones.length
  const complete = completedCount === totalCount
  const currentMilestone = currentIndex >= 0 ? milestones[currentIndex] : null

  return {
    complete,
    completedCount,
    totalCount,
    progressPercent: Math.round((completedCount / totalCount) * 100),
    title: complete
      ? `${snapshot.selectedProjectName || 'This workspace'} is ready for ticket-driven automation.`
      : `${completedCount}/${totalCount} onboarding milestones unlocked`,
    description: complete
      ? 'Organization, project, workflow lane, ticket queue, and live telemetry are all active in the same control plane.'
      : (currentMilestone?.description ?? 'OpenASE is still unlocking core onboarding milestones.'),
    actionLabel: complete
      ? 'Next focus: keep routing tickets and refine workflows as real work lands.'
      : (currentMilestone?.action ?? 'Continue onboarding setup in the current workspace.'),
    stats: [
      { label: 'Organizations', value: String(snapshot.organizationCount) },
      { label: 'Projects', value: String(snapshot.projectCount) },
      { label: 'Workflows', value: String(snapshot.workflowCount) },
      { label: 'Tickets', value: String(snapshot.ticketCount) },
      { label: 'Agents', value: String(snapshot.agentCount) },
    ],
    milestones,
  }
}
