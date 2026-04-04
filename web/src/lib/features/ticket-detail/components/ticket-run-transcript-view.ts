import { formatRelativeTime } from '$lib/utils'
import type { StreamConnectionState } from '$lib/api/sse'
import type { TicketRun, TicketRunTranscriptBlock } from '../types'

export function statusLabel(run: TicketRun) {
  if (run.status === 'completed') return 'Completed'
  if (run.status === 'failed') return 'Failed'
  if (run.status === 'ended') return 'Ended'
  if ((run.currentStepStatus ?? '').toLowerCase().includes('approval')) return 'Awaiting Approval'
  if ((run.currentStepStatus ?? '').toLowerCase().includes('input')) return 'Waiting Input'
  if (run.status === 'launching') return 'Launching'
  return 'Running'
}

export function statusTone(run: TicketRun) {
  switch (run.status) {
    case 'completed':
      return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700'
    case 'failed':
      return 'border-red-500/20 bg-red-500/10 text-red-700'
    case 'ended':
      return 'border-slate-500/20 bg-slate-500/10 text-slate-700'
    case 'ready':
    case 'executing':
      return 'border-sky-500/20 bg-sky-500/10 text-sky-700'
    default:
      return 'border-amber-500/20 bg-amber-500/10 text-amber-700'
  }
}

export function blockLabel(block: TicketRunTranscriptBlock) {
  switch (block.kind) {
    case 'assistant_message':
      return 'Assistant'
    case 'terminal_output':
      return 'Terminal'
    case 'tool_call':
      return 'Tool'
    case 'task_status':
      return 'Status'
    case 'diff':
      return 'Diff'
    case 'step':
      return 'Step'
    case 'phase':
      return 'Phase'
    case 'interrupt':
      return 'Interrupt'
    case 'result':
      return 'Result'
  }
}

export function blockCardClass(block: TicketRunTranscriptBlock) {
  switch (block.kind) {
    case 'assistant_message':
      return 'border-border bg-background'
    case 'terminal_output':
      return 'border-slate-400/20 bg-slate-950 text-slate-50'
    case 'tool_call':
      return 'border-sky-500/20 bg-sky-500/5'
    case 'task_status':
      return block.statusType === 'reasoning_updated'
        ? 'border-amber-500/20 bg-amber-500/5'
        : 'border-sky-500/20 bg-sky-500/5'
    case 'diff':
      return 'border-sky-500/20 bg-sky-500/5'
    case 'interrupt':
      return 'border-amber-400/30 bg-amber-50/80'
    case 'result':
      switch (block.outcome) {
        case 'completed':
          return 'border-emerald-500/20 bg-emerald-500/5'
        case 'failed':
          return 'border-red-500/20 bg-red-500/5'
        default:
          return 'border-amber-500/20 bg-amber-500/5'
      }
    default:
      return 'border-border bg-muted/25'
  }
}

export function blockMeta(block: TicketRunTranscriptBlock) {
  switch (block.kind) {
    case 'phase':
      return block.phase
    case 'step':
      return block.stepStatus
    case 'tool_call':
      return block.toolName
    case 'task_status':
      return block.statusType.replace(/_/g, ' ')
    case 'diff':
      return block.diff.file
    case 'interrupt':
      return block.interruptKind.replace(/_/g, ' ')
    case 'result':
      return block.outcome
    default:
      return ''
  }
}

export function blockTimestamp(block: TicketRunTranscriptBlock) {
  if (
    block.kind === 'phase' ||
    block.kind === 'step' ||
    block.kind === 'tool_call' ||
    block.kind === 'task_status' ||
    block.kind === 'diff' ||
    block.kind === 'interrupt'
  ) {
    return formatRelativeTime(block.at)
  }
  return ''
}

export function blockMutedTextClass(block: TicketRunTranscriptBlock) {
  return block.kind === 'terminal_output' ? 'text-slate-400' : 'text-muted-foreground'
}

export function blockLabelClass(block: TicketRunTranscriptBlock) {
  return block.kind === 'terminal_output'
    ? 'font-medium uppercase tracking-[0.14em] text-slate-300'
    : 'text-muted-foreground font-medium uppercase tracking-[0.14em]'
}

export function connectionLabel(
  streamState: StreamConnectionState,
  recovering: boolean,
  liveSelected: boolean,
) {
  if (recovering) return 'Recovering transcript'
  switch (streamState) {
    case 'connecting':
      return 'Connecting'
    case 'retrying':
      return 'Reconnecting'
    case 'live':
      return liveSelected ? 'Live stream' : 'Connected'
    default:
      return ''
  }
}

export function connectionTone(streamState: StreamConnectionState, recovering: boolean) {
  if (recovering || streamState === 'retrying') {
    return 'border-amber-400/30 bg-amber-500/10 text-amber-700'
  }
  if (streamState === 'live') {
    return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
  }
  return 'border-border bg-muted/40 text-muted-foreground'
}
