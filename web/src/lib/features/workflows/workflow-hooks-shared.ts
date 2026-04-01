export const workflowHookEvents = ['on_activate', 'on_reload'] as const
export const ticketHookEvents = [
  'on_claim',
  'on_start',
  'on_complete',
  'on_done',
  'on_error',
  'on_cancel',
] as const
export const workflowHookFailurePolicies = ['block', 'warn', 'ignore'] as const

export type WorkflowHookEvent = (typeof workflowHookEvents)[number]
export type TicketHookEvent = (typeof ticketHookEvents)[number]
export type WorkflowHookFailurePolicy = (typeof workflowHookFailurePolicies)[number]

export type WorkflowHookRowDraft = {
  id: string
  cmd: string
  timeout: string
  onFailure: WorkflowHookFailurePolicy
  workdir: string
}

export type WorkflowHookRowDraftErrors = {
  cmd?: string
  timeout?: string
  onFailure?: string
}

export type WorkflowHooksDraft = {
  workflowHooks: Record<WorkflowHookEvent, WorkflowHookRowDraft[]>
  ticketHooks: Record<TicketHookEvent, WorkflowHookRowDraft[]>
}

export type WorkflowHookCommandPayload = {
  cmd: string
  timeout?: number
  on_failure: WorkflowHookFailurePolicy
}

export type TicketHookCommandPayload = WorkflowHookCommandPayload & {
  workdir?: string
}

export type WorkflowHooksPayload = {
  workflow_hooks?: Partial<Record<WorkflowHookEvent, WorkflowHookCommandPayload[]>>
  ticket_hooks?: Partial<Record<TicketHookEvent, TicketHookCommandPayload[]>>
}

export type WorkflowHookEventOption<TEvent extends string> = {
  event: TEvent
  label: string
  description: string
}

export type WorkflowHookDraftValidation = {
  rowErrors: Record<string, WorkflowHookRowDraftErrors>
  hasErrors: boolean
  firstError: string
}

export const workflowHookEventOptions: WorkflowHookEventOption<WorkflowHookEvent>[] = [
  {
    event: 'on_activate',
    label: 'On activate',
    description: 'Run when the workflow is activated.',
  },
  {
    event: 'on_reload',
    label: 'On reload',
    description: 'Run when a new workflow version is published.',
  },
]

export const ticketHookEventOptions: WorkflowHookEventOption<TicketHookEvent>[] = [
  {
    event: 'on_claim',
    label: 'On claim',
    description: 'Prepare the ticket workspace before the agent starts.',
  },
  {
    event: 'on_start',
    label: 'On start',
    description: 'Check runtime prerequisites just before agent launch.',
  },
  {
    event: 'on_complete',
    label: 'On complete',
    description: 'Gate successful completion before the finish state transition.',
  },
  {
    event: 'on_done',
    label: 'On done',
    description: 'Run non-blocking cleanup after the ticket reaches a finish state.',
  },
  {
    event: 'on_error',
    label: 'On error',
    description: 'Run after a failed attempt before the next retry decision.',
  },
  {
    event: 'on_cancel',
    label: 'On cancel',
    description: 'Run non-blocking cleanup when a ticket is manually canceled.',
  },
]

export function listWorkflowHookEventOptions() {
  return workflowHookEventOptions
}

export function listTicketHookEventOptions() {
  return ticketHookEventOptions
}

export function listWorkflowHookFailurePolicies() {
  return workflowHookFailurePolicies
}

export function createEmptyHookGroup<TEvent extends string>(events: readonly TEvent[]) {
  return Object.fromEntries(events.map((event) => [event, []])) as unknown as Record<
    TEvent,
    WorkflowHookRowDraft[]
  >
}

export function omitSupportedEvents<TEvent extends string>(
  value: Record<string, unknown> | undefined,
  events: readonly TEvent[],
) {
  if (!value) {
    return undefined
  }

  const remaining = Object.fromEntries(
    Object.entries(value).filter(([key]) => !events.includes(key as TEvent)),
  )

  return Object.keys(remaining).length > 0 ? remaining : undefined
}

export function isEffectivelyEmptyRow(row: WorkflowHookRowDraft, allowWorkdir: boolean) {
  return (
    row.cmd.trim() === '' &&
    row.timeout.trim() === '' &&
    row.onFailure === 'block' &&
    (!allowWorkdir || row.workdir.trim() === '')
  )
}

export function asObject(value: unknown) {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

export function readString(object: Record<string, unknown>, key: string) {
  return typeof object[key] === 'string' ? object[key] : undefined
}

export function readNumberAsString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0) {
    return String(value)
  }
  if (typeof value === 'string') {
    return value
  }
  return ''
}

export function readFailurePolicy(object: Record<string, unknown>): WorkflowHookFailurePolicy {
  const value = object.on_failure
  return typeof value === 'string' &&
    workflowHookFailurePolicies.includes(value as WorkflowHookFailurePolicy)
    ? (value as WorkflowHookFailurePolicy)
    : 'block'
}
