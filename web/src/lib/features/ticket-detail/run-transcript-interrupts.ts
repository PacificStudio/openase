import type {
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptInterruptOption,
} from './types'
import { i18nStore } from '$lib/i18n/store.svelte'
import { readPayloadString } from './run-transcript-blocks'

export function buildInterruptBlock(
  entry: TicketRunTraceEntry,
): Extract<TicketRunTranscriptBlock, { kind: 'interrupt' }> {
  const options = parseInterruptOptions(entry.payload.options)
  const interruptKind =
    entry.kind === 'user_input_requested'
      ? 'user_input'
      : normalizeApprovalInterruptKind(readPayloadString(entry.payload, 'kind'))

  return {
    kind: 'interrupt',
    id: `interrupt:${readPayloadString(entry.payload, 'request_id') || entry.id}`,
    interruptKind,
    title: interruptTitle(interruptKind),
    summary: entry.output || interruptSummary(interruptKind, entry.payload),
    at: entry.createdAt,
    payload: entry.payload,
    options,
  }
}

function parseInterruptOptions(value: unknown): TicketRunTranscriptInterruptOption[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => (item && typeof item === 'object' ? (item as Record<string, unknown>) : null))
    .filter((item): item is Record<string, unknown> => item != null)
    .map((item) => ({
      id: typeof item.id === 'string' ? item.id : '',
      label:
        typeof item.label === 'string'
          ? item.label
          : i18nStore.t('ticketDetail.interrupt.decision'),
      rawDecision: typeof item.raw_decision === 'string' ? item.raw_decision : undefined,
    }))
    .filter((item) => item.id !== '')
}

function normalizeApprovalInterruptKind(raw: string) {
  return raw === 'file_change' ? 'file_change_approval' : 'command_execution_approval'
}

function interruptTitle(kind: string) {
  switch (kind) {
    case 'user_input':
      return i18nStore.t('ticketDetail.interrupt.userInputRequired')
    case 'file_change_approval':
      return i18nStore.t('ticketDetail.interrupt.fileChangeApprovalRequired')
    default:
      return i18nStore.t('ticketDetail.interrupt.commandApprovalRequired')
  }
}

function interruptSummary(kind: string, payload: Record<string, unknown>) {
  const questions = payload.questions
  if (kind === 'user_input' && Array.isArray(questions) && questions.length > 0) {
    const first = questions[0]
    if (
      first &&
      typeof first === 'object' &&
      typeof (first as Record<string, unknown>).question === 'string'
    ) {
      return String((first as Record<string, unknown>).question)
    }
  }

  if (kind === 'file_change_approval') {
    return (
      readInterruptString(payload, 'file', 'path', 'target') ||
      i18nStore.t('ticketDetail.interrupt.pendingFileApproval')
    )
  }

  return (
    readInterruptString(payload, 'command') || i18nStore.t('ticketDetail.interrupt.pendingApproval')
  )
}

function readInterruptString(payload: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = payload[key]
    if (typeof value === 'string' && value.trim()) {
      return value
    }
  }
  return ''
}
