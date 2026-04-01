import type {
  TicketRun,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptState,
} from './types'

type TimelineEvent =
  | { kind: 'step'; entry: TicketRunStepEntry }
  | { kind: 'trace'; entry: TicketRunTraceEntry }

export function sortTicketRuns(runs: TicketRun[]): TicketRun[] {
  return runs
    .slice()
    .sort((left, right) =>
      right.attemptNumber !== left.attemptNumber
        ? right.attemptNumber - left.attemptNumber
        : Date.parse(right.createdAt) - Date.parse(left.createdAt),
    )
}

export function mergeRun(runs: TicketRun[], run: TicketRun): TicketRun[] {
  const nextRuns = runs.filter((item) => item.id !== run.id)
  nextRuns.push(run)
  return sortTicketRuns(nextRuns)
}

export function shouldSwitchToRun(currentRun: TicketRun | null, incomingRun: TicketRun): boolean {
  if (!currentRun) {
    return true
  }
  if (currentRun.id === incomingRun.id) {
    return true
  }
  if (incomingRun.attemptNumber !== currentRun.attemptNumber) {
    return incomingRun.attemptNumber > currentRun.attemptNumber
  }
  return Date.parse(incomingRun.createdAt) > Date.parse(currentRun.createdAt)
}

export function buildRunTimeline(
  steps: TicketRunStepEntry[],
  traces: TicketRunTraceEntry[],
): TimelineEvent[] {
  const items: TimelineEvent[] = [
    ...steps.map((entry) => ({ kind: 'step' as const, entry })),
    ...traces.map((entry) => ({ kind: 'trace' as const, entry })),
  ]

  items.sort((left, right) => {
    const timeDiff = Date.parse(left.entry.createdAt) - Date.parse(right.entry.createdAt)
    if (timeDiff !== 0) {
      return timeDiff
    }

    if (left.kind === 'trace' && right.kind === 'trace') {
      return left.entry.sequence - right.entry.sequence
    }

    return left.entry.id.localeCompare(right.entry.id)
  })

  return items
}

export function mergeRunTextBlock(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
  blockKind: 'assistant_message' | 'terminal_output',
): TicketRunTranscriptState {
  const itemId = readPayloadString(entry.payload, 'item_id') || undefined
  const blockID = `${blockKind}:${itemId ?? entry.stream}`
  const existingIndex = state.blocks.findIndex(
    (block) =>
      block.kind === blockKind && block.id === blockID && (itemId ? block.itemId === itemId : true),
  )
  const isSnapshot = entry.kind.endsWith('_snapshot')

  if (existingIndex === -1) {
    return {
      ...state,
      blocks: [
        ...state.blocks,
        {
          kind: blockKind,
          id: blockID,
          itemId,
          text: entry.output,
          streaming: !isTerminalRunStatus(state.currentRun?.status),
        },
      ],
    }
  }

  const existing = state.blocks[existingIndex]
  if (existing.kind !== blockKind) {
    return state
  }

  const nextText = isSnapshot ? entry.output : `${existing.text}${entry.output}`
  const nextBlocks = state.blocks.slice()
  nextBlocks[existingIndex] = {
    ...existing,
    text: nextText,
    streaming: !isTerminalRunStatus(state.currentRun?.status),
  }

  return {
    ...state,
    blocks: nextBlocks,
  }
}

export function seedRunBlocks(run: TicketRun): TicketRunTranscriptBlock[] {
  const blocks: TicketRunTranscriptBlock[] = [
    {
      kind: 'phase',
      id: `phase:launching:${run.createdAt}`,
      phase: 'launching',
      at: run.createdAt,
      summary: 'Run created.',
    },
  ]

  if (run.runtimeStartedAt) {
    blocks.push({
      kind: 'phase',
      id: `phase:ready:${run.runtimeStartedAt}`,
      phase: 'ready',
      at: run.runtimeStartedAt,
      summary: 'Runtime ready.',
    })
  }

  if (run.status === 'executing') {
    blocks.push({
      kind: 'phase',
      id: `phase:executing:${run.runtimeStartedAt ?? run.createdAt}`,
      phase: 'executing',
      at: run.runtimeStartedAt ?? run.createdAt,
      summary: run.currentStepSummary || 'Run executing.',
    })
  }

  if (run.status === 'completed' || run.status === 'failed' || run.status === 'stalled') {
    blocks.push(buildResultBlock(run))
  }

  return dedupeBlocks(blocks)
}

export function buildLifecycleBlock(
  lifecycle: TicketRunLifecycleEvent,
): TicketRunTranscriptBlock | null {
  switch (lifecycle.eventType) {
    case 'agent.claimed':
    case 'agent.launching':
      return {
        kind: 'phase',
        id: `phase:${lifecycle.eventType}:${lifecycle.createdAt}`,
        phase: 'launching',
        at: lifecycle.createdAt,
        summary: lifecycle.message,
      }
    case 'agent.ready':
      return {
        kind: 'phase',
        id: `phase:${lifecycle.eventType}:${lifecycle.createdAt}`,
        phase: 'ready',
        at: lifecycle.createdAt,
        summary: lifecycle.message,
      }
    case 'agent.paused':
      return {
        kind: 'phase',
        id: `phase:${lifecycle.eventType}:${lifecycle.createdAt}`,
        phase: 'paused',
        at: lifecycle.createdAt,
        summary: lifecycle.message,
      }
    case 'agent.failed':
      return {
        kind: 'result',
        id: `result:${lifecycle.eventType}:${lifecycle.createdAt}`,
        outcome: 'failed',
        summary: lifecycle.message,
      }
    case 'agent.completed':
      return {
        kind: 'result',
        id: `result:${lifecycle.eventType}:${lifecycle.createdAt}`,
        outcome: 'completed',
        summary: lifecycle.message,
      }
    case 'agent.terminated':
      return {
        kind: 'result',
        id: `result:${lifecycle.eventType}:${lifecycle.createdAt}`,
        outcome: 'stalled',
        summary: lifecycle.message,
      }
    default:
      return null
  }
}

export function finalizeTerminalRunBlocks(
  state: TicketRunTranscriptState,
): TicketRunTranscriptState {
  if (!isTerminalRunStatus(state.currentRun?.status)) {
    return state
  }

  return {
    ...state,
    blocks: state.blocks.map((block) =>
      block.kind === 'assistant_message' || block.kind === 'terminal_output'
        ? { ...block, streaming: false }
        : block,
    ),
  }
}

export function isTerminalRunStatus(status: TicketRun['status'] | undefined): boolean {
  return status === 'completed' || status === 'failed' || status === 'stalled'
}

export function readPayloadString(payload: Record<string, unknown>, key: string): string {
  const value = payload[key]
  return typeof value === 'string' ? value : ''
}

export function hasBlock(blocks: TicketRunTranscriptBlock[], id: string): boolean {
  return blocks.some((block) => block.id === id)
}

function buildResultBlock(run: TicketRun): TicketRunTranscriptBlock {
  return {
    kind: 'result',
    id: `result:${run.status}:${run.id}`,
    outcome:
      run.status === 'completed' ? 'completed' : run.status === 'failed' ? 'failed' : 'stalled',
    summary:
      run.lastError ||
      run.currentStepSummary ||
      (run.status === 'completed'
        ? 'Run completed.'
        : run.status === 'failed'
          ? 'Run failed.'
          : 'Run stalled.'),
  }
}

function dedupeBlocks(blocks: TicketRunTranscriptBlock[]): TicketRunTranscriptBlock[] {
  const seen = new Set<string>()
  return blocks.filter((block) => {
    if (seen.has(block.id)) {
      return false
    }
    seen.add(block.id)
    return true
  })
}
