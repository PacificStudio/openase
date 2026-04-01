export {
  listTicketHookEventOptions,
  listWorkflowHookEventOptions,
  listWorkflowHookFailurePolicies,
  ticketHookEvents,
  workflowHookEvents,
  workflowHookFailurePolicies,
  type TicketHookCommandPayload,
  type TicketHookEvent,
  type WorkflowHookCommandPayload,
  type WorkflowHookDraftValidation,
  type WorkflowHookEvent,
  type WorkflowHookEventOption,
  type WorkflowHookFailurePolicy,
  type WorkflowHookRowDraft,
  type WorkflowHookRowDraftErrors,
  type WorkflowHooksDraft,
  type WorkflowHooksPayload,
} from './workflow-hooks-shared'

import {
  asObject,
  createEmptyHookGroup,
  omitSupportedEvents,
  readFailurePolicy,
  readNumberAsString,
  readString,
  ticketHookEventOptions,
  ticketHookEvents,
  workflowHookEventOptions,
  workflowHookEvents,
  type WorkflowHookDraftValidation,
  type WorkflowHookRowDraft,
  type WorkflowHookRowDraftErrors,
  type WorkflowHooksDraft,
  type WorkflowHooksPayload,
} from './workflow-hooks-shared'
import { serializeHookGroup, summarizeDraftGroup, validateHookRows } from './workflow-hooks-ops'

type ParseResult<T> =
  | { ok: true; value: T; validation: WorkflowHookDraftValidation }
  | { ok: false; error: string; validation: WorkflowHookDraftValidation }

let hookRowSequence = 0

export function createWorkflowHooksDraft(
  rawHooks?: WorkflowHooksPayload | Record<string, unknown> | null,
): WorkflowHooksDraft {
  return {
    workflowHooks: readHookGroup(rawHooks, 'workflow_hooks', workflowHookEvents, false),
    ticketHooks: readHookGroup(rawHooks, 'ticket_hooks', ticketHookEvents, true),
  }
}

export function createWorkflowHookRowDraft(
  partial: Partial<Omit<WorkflowHookRowDraft, 'id'>> = {},
): WorkflowHookRowDraft {
  return {
    id: nextHookRowId(),
    cmd: partial.cmd ?? '',
    timeout: partial.timeout ?? '',
    onFailure: partial.onFailure ?? 'block',
    workdir: partial.workdir ?? '',
  }
}

export function validateWorkflowHooksDraft(draft: WorkflowHooksDraft): WorkflowHookDraftValidation {
  const rowErrors: Record<string, WorkflowHookRowDraftErrors> = {}
  const messages: string[] = []

  for (const option of workflowHookEventOptions) {
    validateHookRows(option.label, draft.workflowHooks[option.event], false, rowErrors, messages)
  }
  for (const option of ticketHookEventOptions) {
    validateHookRows(option.label, draft.ticketHooks[option.event], true, rowErrors, messages)
  }

  return {
    rowErrors,
    hasErrors: messages.length > 0,
    firstError: messages[0] ?? '',
  }
}

export function parseWorkflowHooksDraft(
  draft: WorkflowHooksDraft,
): ParseResult<WorkflowHooksPayload | undefined> {
  const validation = validateWorkflowHooksDraft(draft)
  if (validation.hasErrors) {
    return { ok: false, error: validation.firstError, validation }
  }

  const workflowHooks = serializeHookGroup(draft.workflowHooks, workflowHookEvents, false)
  const ticketHooks = serializeHookGroup(draft.ticketHooks, ticketHookEvents, true)
  const payload: WorkflowHooksPayload = {}

  if (Object.keys(workflowHooks).length > 0) {
    payload.workflow_hooks = workflowHooks
  }
  if (Object.keys(ticketHooks).length > 0) {
    payload.ticket_hooks = ticketHooks
  }

  return {
    ok: true,
    value: Object.keys(payload).length > 0 ? payload : undefined,
    validation,
  }
}

export function mergeWorkflowHooksPayload(
  supportedHooks: WorkflowHooksPayload | undefined,
  rawHooks?: Record<string, unknown> | null,
): Record<string, unknown> | undefined {
  const merged: Record<string, unknown> = {
    ...(extractUnsupportedWorkflowHooks(rawHooks) ?? {}),
  }

  if (supportedHooks?.workflow_hooks && Object.keys(supportedHooks.workflow_hooks).length > 0) {
    merged.workflow_hooks = {
      ...(asObject(merged.workflow_hooks) ?? {}),
      ...supportedHooks.workflow_hooks,
    }
  }
  if (supportedHooks?.ticket_hooks && Object.keys(supportedHooks.ticket_hooks).length > 0) {
    merged.ticket_hooks = {
      ...(asObject(merged.ticket_hooks) ?? {}),
      ...supportedHooks.ticket_hooks,
    }
  }

  return Object.keys(merged).length > 0 ? merged : undefined
}

export function readWorkflowHooksPayload(
  rawHooks?: Record<string, unknown> | null,
): WorkflowHooksPayload | undefined {
  const parsed = parseWorkflowHooksDraft(createWorkflowHooksDraft(rawHooks))
  return parsed.ok ? parsed.value : undefined
}

export function workflowHooksDraftSignature(draft: WorkflowHooksDraft) {
  return JSON.stringify({
    workflowHooks: summarizeDraftGroup(draft.workflowHooks, workflowHookEvents, false),
    ticketHooks: summarizeDraftGroup(draft.ticketHooks, ticketHookEvents, true),
  })
}

function nextHookRowId() {
  hookRowSequence += 1
  return `workflow-hook-row-${hookRowSequence}`
}

function readHookGroup<TEvent extends string>(
  rawHooks: WorkflowHooksPayload | Record<string, unknown> | null | undefined,
  key: 'workflow_hooks' | 'ticket_hooks',
  events: readonly TEvent[],
  allowWorkdir: boolean,
) {
  const group = createEmptyHookGroup(events)
  const sourceGroup = asObject(asObject(rawHooks)?.[key])
  if (!sourceGroup) {
    return group
  }

  for (const event of events) {
    const rows = Array.isArray(sourceGroup[event]) ? sourceGroup[event] : []
    group[event] = rows.flatMap((row) => {
      const rowObject = asObject(row)
      if (!rowObject) {
        return []
      }

      return [
        createWorkflowHookRowDraft({
          cmd: readString(rowObject, 'cmd') ?? '',
          timeout: readNumberAsString(rowObject, 'timeout'),
          onFailure: readFailurePolicy(rowObject),
          workdir: allowWorkdir ? (readString(rowObject, 'workdir') ?? '') : '',
        }),
      ]
    })
  }

  return group
}

function extractUnsupportedWorkflowHooks(rawHooks?: Record<string, unknown> | null) {
  const source = asObject(rawHooks)
  if (!source) {
    return undefined
  }

  const unsupported: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(source)) {
    if (key === 'workflow_hooks') {
      const remaining = omitSupportedEvents(asObject(value), workflowHookEvents)
      if (remaining) {
        unsupported[key] = remaining
      }
      continue
    }

    if (key === 'ticket_hooks') {
      const remaining = omitSupportedEvents(asObject(value), ticketHookEvents)
      if (remaining) {
        unsupported[key] = remaining
      }
      continue
    }

    unsupported[key] = value
  }

  return Object.keys(unsupported).length > 0 ? unsupported : undefined
}
