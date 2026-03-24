import type { ScheduledJob } from '$lib/api/contracts'

export type ScheduledJobDraft = {
  name: string
  cronExpression: string
  workflowId: string
  isEnabled: boolean
  ticketTitle: string
  ticketDescription: string
  ticketPriority: string
  ticketType: string
  ticketBudgetUsd: string
  ticketCreatedBy: string
}

export const scheduledJobPriorityOptions = ['urgent', 'high', 'medium', 'low'] as const
export const scheduledJobTypeOptions = ['feature', 'bugfix', 'refactor', 'chore'] as const

export function emptyScheduledJobDraft(defaultWorkflowId: string): ScheduledJobDraft {
  return {
    name: '',
    cronExpression: '',
    workflowId: defaultWorkflowId,
    isEnabled: true,
    ticketTitle: '',
    ticketDescription: '',
    ticketPriority: 'medium',
    ticketType: 'feature',
    ticketBudgetUsd: '',
    ticketCreatedBy: '',
  }
}

export function scheduledJobDraftFromRecord(
  job: ScheduledJob,
  defaultWorkflowId: string,
): ScheduledJobDraft {
  return {
    name: job.name,
    cronExpression: job.cron_expression,
    workflowId: job.workflow_id || defaultWorkflowId,
    isEnabled: job.is_enabled,
    ticketTitle: job.ticket_template.title ?? '',
    ticketDescription: job.ticket_template.description ?? '',
    ticketPriority: job.ticket_template.priority ?? 'medium',
    ticketType: job.ticket_template.type ?? 'feature',
    ticketBudgetUsd:
      job.ticket_template.budget_usd > 0 ? String(job.ticket_template.budget_usd) : '',
    ticketCreatedBy: job.ticket_template.created_by ?? '',
  }
}

export function parseScheduledJobDraft(value: ScheduledJobDraft) {
  const name = value.name.trim()
  const cronExpression = value.cronExpression.trim()
  const workflowId = value.workflowId.trim()
  const ticketTitle = value.ticketTitle.trim()
  const ticketDescription = value.ticketDescription.trim()
  const ticketCreatedBy = value.ticketCreatedBy.trim()
  const ticketBudgetRaw = value.ticketBudgetUsd.trim()

  if (!name) {
    return { ok: false as const, error: 'Job name is required.' }
  }
  if (!cronExpression) {
    return { ok: false as const, error: 'Cron expression is required.' }
  }
  if (!workflowId) {
    return { ok: false as const, error: 'Workflow selection is required.' }
  }

  let budgetUsd: number | undefined
  if (ticketBudgetRaw) {
    const parsedBudget = Number(ticketBudgetRaw)
    if (!Number.isFinite(parsedBudget) || parsedBudget < 0) {
      return { ok: false as const, error: 'Ticket budget must be a non-negative number.' }
    }
    budgetUsd = parsedBudget
  }

  const ticket_template: {
    budget_usd?: number
    created_by?: string
    description?: string
    priority?: string
    title?: string
    type?: string
  } = {}

  if (ticketTitle) ticket_template.title = ticketTitle
  if (ticketDescription) ticket_template.description = ticketDescription
  if (ticketCreatedBy) ticket_template.created_by = ticketCreatedBy
  if (budgetUsd != null) ticket_template.budget_usd = budgetUsd
  if (value.ticketPriority) ticket_template.priority = value.ticketPriority
  if (value.ticketType) ticket_template.type = value.ticketType

  return {
    ok: true as const,
    value: {
      name,
      cron_expression: cronExpression,
      workflow_id: workflowId,
      is_enabled: value.isEnabled,
      ticket_template,
    },
  }
}
