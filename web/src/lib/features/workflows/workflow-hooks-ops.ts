import {
  isEffectivelyEmptyRow,
  workflowHookFailurePolicies,
  type TicketHookCommandPayload,
  type WorkflowHookCommandPayload,
  type WorkflowHookRowDraft,
  type WorkflowHookRowDraftErrors,
} from './workflow-hooks-shared'

export function validateHookRows(
  eventLabel: string,
  rows: WorkflowHookRowDraft[],
  allowWorkdir: boolean,
  rowErrors: Record<string, WorkflowHookRowDraftErrors>,
  messages: string[],
) {
  for (const row of rows) {
    if (isEffectivelyEmptyRow(row, allowWorkdir)) {
      continue
    }

    const errors: WorkflowHookRowDraftErrors = {}
    appendCommandError(row, eventLabel, errors, messages)
    appendTimeoutError(row, eventLabel, errors, messages)

    if (!workflowHookFailurePolicies.includes(row.onFailure)) {
      errors.onFailure = 'Failure policy must be block, warn, or ignore.'
      messages.push(`${eventLabel}: failure policy must be block, warn, or ignore.`)
    }

    if (Object.keys(errors).length > 0) {
      rowErrors[row.id] = errors
    }
  }
}

export function serializeHookGroup<TEvent extends string>(
  group: Record<TEvent, WorkflowHookRowDraft[]>,
  events: readonly TEvent[],
  allowWorkdir: boolean,
) {
  const serialized: Partial<
    Record<TEvent, Array<WorkflowHookCommandPayload | TicketHookCommandPayload>>
  > = {}

  for (const event of events) {
    const rows = group[event]
      .filter((row) => !isEffectivelyEmptyRow(row, allowWorkdir))
      .map((row) => serializeHookRow(row, allowWorkdir))

    if (rows.length > 0) {
      serialized[event] = rows
    }
  }

  return serialized
}

export function summarizeDraftGroup<TEvent extends string>(
  group: Record<TEvent, WorkflowHookRowDraft[]>,
  events: readonly TEvent[],
  allowWorkdir: boolean,
) {
  const summarized: Partial<Record<TEvent, Array<Record<string, string>>>> = {}

  for (const event of events) {
    const rows = group[event]
      .map((row) => ({
        cmd: row.cmd.trim(),
        timeout: row.timeout.trim(),
        onFailure: row.onFailure,
        ...(allowWorkdir ? { workdir: row.workdir.trim() } : {}),
      }))
      .filter(
        (row) =>
          !(
            row.cmd === '' &&
            row.timeout === '' &&
            row.onFailure === 'block' &&
            (!allowWorkdir || row.workdir === '')
          ),
      )

    if (rows.length > 0) {
      summarized[event] = rows
    }
  }

  return summarized
}

function appendCommandError(
  row: WorkflowHookRowDraft,
  eventLabel: string,
  errors: WorkflowHookRowDraftErrors,
  messages: string[],
) {
  if (row.cmd.trim()) {
    return
  }

  errors.cmd = 'Command is required.'
  messages.push(`${eventLabel}: command is required.`)
}

function appendTimeoutError(
  row: WorkflowHookRowDraft,
  eventLabel: string,
  errors: WorkflowHookRowDraftErrors,
  messages: string[],
) {
  const timeout = row.timeout.trim()
  if (!timeout) {
    return
  }

  if (!/^\d+$/.test(timeout)) {
    errors.timeout = 'Timeout must be a whole number.'
    messages.push(`${eventLabel}: timeout must be a whole number.`)
    return
  }

  if (Number.parseInt(timeout, 10) >= 0) {
    return
  }

  errors.timeout = 'Timeout must be zero or greater.'
  messages.push(`${eventLabel}: timeout must be zero or greater.`)
}

function serializeHookRow(row: WorkflowHookRowDraft, allowWorkdir: boolean) {
  const payload: WorkflowHookCommandPayload | TicketHookCommandPayload = {
    cmd: row.cmd.trim(),
    on_failure: row.onFailure,
  }
  const timeout = row.timeout.trim()
  if (timeout) {
    payload.timeout = Number.parseInt(timeout, 10)
  }
  if (allowWorkdir) {
    const workdir = row.workdir.trim()
    if (workdir) {
      ;(payload as TicketHookCommandPayload).workdir = workdir
    }
  }
  return payload
}
