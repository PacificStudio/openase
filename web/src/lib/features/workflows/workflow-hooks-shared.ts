import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'

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

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

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

type WorkflowHookEventOptionTemplate<TEvent extends string> = {
  event: TEvent
  labelKey: TranslationKey
  descriptionKey: TranslationKey
}

const workflowHookEventOptionTemplates: WorkflowHookEventOptionTemplate<WorkflowHookEvent>[] = [
  {
    event: 'on_activate',
    labelKey: 'workflowHook.event.onActivate.label',
    descriptionKey: 'workflowHook.event.onActivate.description',
  },
  {
    event: 'on_reload',
    labelKey: 'workflowHook.event.onReload.label',
    descriptionKey: 'workflowHook.event.onReload.description',
  },
]

const ticketHookEventOptionTemplates: WorkflowHookEventOptionTemplate<TicketHookEvent>[] = [
  {
    event: 'on_claim',
    labelKey: 'workflowHook.event.onClaim.label',
    descriptionKey: 'workflowHook.event.onClaim.description',
  },
  {
    event: 'on_start',
    labelKey: 'workflowHook.event.onStart.label',
    descriptionKey: 'workflowHook.event.onStart.description',
  },
  {
    event: 'on_complete',
    labelKey: 'workflowHook.event.onComplete.label',
    descriptionKey: 'workflowHook.event.onComplete.description',
  },
  {
    event: 'on_done',
    labelKey: 'workflowHook.event.onDone.label',
    descriptionKey: 'workflowHook.event.onDone.description',
  },
  {
    event: 'on_error',
    labelKey: 'workflowHook.event.onError.label',
    descriptionKey: 'workflowHook.event.onError.description',
  },
  {
    event: 'on_cancel',
    labelKey: 'workflowHook.event.onCancel.label',
    descriptionKey: 'workflowHook.event.onCancel.description',
  },
]

export const workflowHookEventOptions: WorkflowHookEventOption<WorkflowHookEvent>[] =
  workflowHookEventOptionTemplates.map((template) => ({
    event: template.event,
    get label() {
      return translateRaw(template.labelKey)
    },
    get description() {
      return translateRaw(template.descriptionKey)
    },
  }))

export const ticketHookEventOptions: WorkflowHookEventOption<TicketHookEvent>[] =
  ticketHookEventOptionTemplates.map((template) => ({
    event: template.event,
    get label() {
      return translateRaw(template.labelKey)
    },
    get description() {
      return translateRaw(template.descriptionKey)
    },
  }))

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
