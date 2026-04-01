import type { ScheduledJob } from '$lib/api/contracts'
import type { WorkflowStatusOption } from '$lib/features/workflows'

export type ScheduledJobDraft = {
  name: string
  cronExpression: string
  isEnabled: boolean
  ticketStatusId: string
  ticketTitle: string
  ticketDescription: string
  ticketPriority: string
  ticketType: string
  ticketBudgetUsd: string
  ticketCreatedBy: string
}

export const scheduledJobPriorityOptions = ['urgent', 'high', 'medium', 'low'] as const
export const scheduledJobTypeOptions = ['feature', 'bugfix', 'refactor', 'chore'] as const

export function emptyScheduledJobDraft(defaultStatusId: string): ScheduledJobDraft {
  return {
    name: '',
    cronExpression: '',
    isEnabled: true,
    ticketStatusId: defaultStatusId,
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
  statuses: WorkflowStatusOption[],
): ScheduledJobDraft {
  const statusId =
    statuses.find((status) => status.name === (job.ticket_template.status ?? ''))?.id ??
    statuses[0]?.id ??
    ''

  return {
    name: job.name,
    cronExpression: job.cron_expression,
    isEnabled: job.is_enabled,
    ticketStatusId: statusId,
    ticketTitle: job.ticket_template.title ?? '',
    ticketDescription: job.ticket_template.description ?? '',
    ticketPriority: job.ticket_template.priority ?? 'medium',
    ticketType: job.ticket_template.type ?? 'feature',
    ticketBudgetUsd:
      job.ticket_template.budget_usd > 0 ? String(job.ticket_template.budget_usd) : '',
    ticketCreatedBy: job.ticket_template.created_by ?? '',
  }
}

export function parseScheduledJobDraft(value: ScheduledJobDraft, statuses: WorkflowStatusOption[]) {
  const name = value.name.trim()
  const cronExpression = value.cronExpression.trim()
  const ticketStatusId = value.ticketStatusId.trim()
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
  if (!ticketStatusId) {
    return { ok: false as const, error: 'Target status is required.' }
  }
  if (!ticketTitle) {
    return { ok: false as const, error: 'Ticket title is required.' }
  }

  const selectedStatus = statuses.find((status) => status.id === ticketStatusId)
  if (!selectedStatus) {
    return { ok: false as const, error: 'Target status is invalid.' }
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
    status?: string
    title?: string
    type?: string
  } = {}

  if (ticketTitle) ticket_template.title = ticketTitle
  if (ticketDescription) ticket_template.description = ticketDescription
  if (ticketCreatedBy) ticket_template.created_by = ticketCreatedBy
  if (budgetUsd != null) ticket_template.budget_usd = budgetUsd
  ticket_template.status = selectedStatus.name
  if (value.ticketPriority) ticket_template.priority = value.ticketPriority
  if (value.ticketType) ticket_template.type = value.ticketType

  return {
    ok: true as const,
    value: {
      name,
      cron_expression: cronExpression,
      is_enabled: value.isEnabled,
      ticket_template,
    },
  }
}
