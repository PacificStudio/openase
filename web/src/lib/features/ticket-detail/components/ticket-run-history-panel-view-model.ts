import { formatRelativeTime } from '$lib/utils'
import type { TicketRun, TicketRunTranscriptBlock } from '../types'

export const FINISHED_RUN_STATUSES = new Set(['completed', 'failed', 'ended'])

export function ticketRunStatusLabel(run: TicketRun) {
  if (run.status === 'completed') return 'Completed'
  if (run.status === 'failed') return 'Failed'
  if (run.status === 'ended') return 'Ended'
  if ((run.currentStepStatus ?? '').toLowerCase().includes('approval')) return 'Awaiting Approval'
  if ((run.currentStepStatus ?? '').toLowerCase().includes('input')) return 'Waiting Input'
  if (run.status === 'launching') return 'Launching'
  return 'Running'
}

export function ticketRunStatusClass(run: TicketRun) {
  switch (run.status) {
    case 'completed':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600'
    case 'failed':
      return 'border-red-500/30 bg-red-500/10 text-red-600'
    case 'ended':
      return 'border-slate-500/30 bg-slate-500/10 text-slate-600'
    case 'ready':
    case 'executing':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-600'
    default:
      return 'border-amber-500/30 bg-amber-500/10 text-amber-600'
  }
}

export function ticketRunSummaryLine(run: TicketRun) {
  return (
    run.currentStepSummary ||
    run.currentStepStatus ||
    (run.status === 'completed' && run.completedAt
      ? `Completed ${formatRelativeTime(run.completedAt)}`
      : run.status === 'ended' && run.terminalAt
        ? `Ended ${formatRelativeTime(run.terminalAt)}`
        : run.lastHeartbeatAt
          ? `Updated ${formatRelativeTime(run.lastHeartbeatAt)}`
          : `Started ${formatRelativeTime(run.createdAt)}`)
  )
}

export function completionSummaryLabel(run: TicketRun) {
  switch (run.completionSummary?.status) {
    case 'completed':
      return 'Summary'
    case 'failed':
      return 'Summary failed'
    case 'pending':
      return 'Summary pending'
    default:
      return ''
  }
}

export function completionSummaryClass(run: TicketRun) {
  switch (run.completionSummary?.status) {
    case 'completed':
      return 'border-emerald-500/30 bg-emerald-500/5'
    case 'failed':
      return 'border-red-500/30 bg-red-500/5'
    case 'pending':
      return 'border-amber-500/30 bg-amber-500/5'
    default:
      return 'border-border/60 bg-muted/20'
  }
}

export function renderKeyForBlocks(blocks: TicketRunTranscriptBlock[]) {
  const tail = blocks.at(-1)
  if (!tail) return 'empty'
  switch (tail.kind) {
    case 'assistant_message':
    case 'terminal_output':
      return `${tail.id}:${tail.text.length}:${tail.streaming}`
    case 'tool_call':
      return `${tail.id}:${tail.toolName}:${JSON.stringify(tail.arguments ?? null)}`
    case 'task_status':
      return `${tail.id}:${tail.statusType}:${tail.detail ?? ''}`
    case 'diff':
      return `${tail.id}:${tail.diff.file}:${tail.diff.hunks.length}`
    case 'interrupt':
      return `${tail.id}:${tail.summary}:${tail.options.length}`
    case 'phase':
    case 'step':
    case 'result':
      return `${tail.id}:${tail.summary}`
  }
}
