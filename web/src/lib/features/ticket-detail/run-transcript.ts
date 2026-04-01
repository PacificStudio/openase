import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptState,
} from './types'

type TimelineEvent =
  | { kind: 'step'; entry: TicketRunStepEntry }
  | { kind: 'trace'; entry: TicketRunTraceEntry }

export function createEmptyTicketRunTranscriptState(): TicketRunTranscriptState {
  return {
    runs: [],
    currentRun: null,
    blocks: [],
  }
}

export function setTicketRunList(
  state: TicketRunTranscriptState,
  runs: TicketRun[],
): TicketRunTranscriptState {
  const nextRuns = sortRuns(runs)
  const latestRun = nextRuns[0] ?? null

  if (!state.currentRun) {
    return {
      runs: nextRuns,
      currentRun: latestRun,
      blocks: latestRun ? seedRunBlocks(latestRun) : [],
    }
  }

  const matchingCurrent = nextRuns.find((item) => item.id === state.currentRun?.id)
  if (matchingCurrent) {
    return {
      runs: nextRuns,
      currentRun: matchingCurrent,
      blocks: state.blocks.length > 0 ? state.blocks : seedRunBlocks(matchingCurrent),
    }
  }

  return {
    runs: nextRuns,
    currentRun: latestRun,
    blocks: latestRun ? seedRunBlocks(latestRun) : [],
  }
}

export function hydrateTicketRunDetail(
  state: TicketRunTranscriptState,
  detail: TicketRunDetail,
): TicketRunTranscriptState {
  const runs = mergeRun(state.runs, detail.run)
  let nextState: TicketRunTranscriptState = {
    runs,
    currentRun: detail.run,
    blocks: seedRunBlocks(detail.run),
  }

  const timeline = buildTimeline(detail.stepEntries, detail.traceEntries)
  for (const item of timeline) {
    nextState =
      item.kind === 'step'
        ? applyTicketRunStepEntry(nextState, item.entry)
        : applyTicketRunTraceEntry(nextState, item.entry)
  }

  return finalizeTerminalRunBlocks(nextState)
}

export function applyTicketRunStreamFrame(
  state: TicketRunTranscriptState,
  frame: Pick<SSEFrame, 'event' | 'data'>,
): TicketRunTranscriptState {
  try {
    switch (frame.event) {
      case 'ticket.run.lifecycle': {
        const payload = JSON.parse(frame.data) as {
          run: TicketRun
          lifecycle: TicketRunLifecycleEvent
        }
        return applyTicketRunLifecycleEvent(state, payload.run, payload.lifecycle)
      }
      case 'ticket.run.trace': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunTraceEntry }
        return applyTicketRunTraceEntry(state, payload.entry)
      }
      case 'ticket.run.step': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunStepEntry }
        return applyTicketRunStepEntry(state, payload.entry)
      }
      default:
        return state
    }
  } catch {
    return state
  }
}

export function applyTicketRunLifecycleEvent(
  state: TicketRunTranscriptState,
  run: TicketRun,
  lifecycle: TicketRunLifecycleEvent,
): TicketRunTranscriptState {
  const runs = mergeRun(state.runs, run)
  const shouldSwitch = shouldSwitchToRun(state.currentRun, run)
  const currentRun = shouldSwitch ? run : state.currentRun
  const baseBlocks =
    shouldSwitch || !state.currentRun || state.currentRun.id !== run.id
      ? seedRunBlocks(run)
      : state.blocks

  if (!currentRun || currentRun.id !== run.id) {
    return { runs, currentRun, blocks: baseBlocks }
  }

  const nextBlock = buildLifecycleBlock(lifecycle)
  if (!nextBlock || hasBlock(baseBlocks, nextBlock.id)) {
    return finalizeTerminalRunBlocks({ runs, currentRun, blocks: baseBlocks })
  }

  return finalizeTerminalRunBlocks({
    runs,
    currentRun,
    blocks: [...baseBlocks, nextBlock],
  })
}

export function applyTicketRunStepEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  if (state.currentRun?.id !== entry.agentRunId) {
    return state
  }
  if (hasBlock(state.blocks, `step:${entry.id}`)) {
    return state
  }

  return {
    ...state,
    currentRun: {
      ...state.currentRun,
      currentStepStatus: entry.stepStatus,
      currentStepSummary: entry.summary || state.currentRun.currentStepSummary,
    },
    blocks: [
      ...state.blocks,
      {
        kind: 'step',
        id: `step:${entry.id}`,
        stepStatus: entry.stepStatus,
        summary: entry.summary,
        at: entry.createdAt,
      },
    ],
  }
}

export function applyTicketRunTraceEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
): TicketRunTranscriptState {
  if (state.currentRun?.id !== entry.agentRunId) {
    return state
  }

  switch (entry.kind) {
    case 'assistant_delta':
    case 'assistant_snapshot':
      return mergeTextBlock(state, entry, 'assistant_message')
    case 'command_output_delta':
    case 'command_output_snapshot':
      return mergeTextBlock(state, entry, 'terminal_output')
    case 'tool_call_started':
      if (hasBlock(state.blocks, `tool:${entry.id}`)) {
        return state
      }
      return {
        ...state,
        blocks: [
          ...state.blocks,
          {
            kind: 'tool_call',
            id: `tool:${entry.id}`,
            toolName: readPayloadString(entry.payload, 'tool') || entry.output || entry.stream,
            summary: readPayloadString(entry.payload, 'phase') || undefined,
            at: entry.createdAt,
          },
        ],
      }
    default:
      return state
  }
}

function buildTimeline(
  steps: TicketRunStepEntry[],
  traces: TicketRunTraceEntry[],
): TimelineEvent[] {
  const items: TimelineEvent[] = [
    ...steps.map((entry) => ({ kind: 'step' as const, entry })),
    ...traces.map((entry) => ({ kind: 'trace' as const, entry })),
  ]

  items.sort((left, right) => {
    const leftAt = left.kind === 'step' ? left.entry.createdAt : left.entry.createdAt
    const rightAt = right.kind === 'step' ? right.entry.createdAt : right.entry.createdAt
    const timeDiff = Date.parse(leftAt) - Date.parse(rightAt)
    if (timeDiff !== 0) {
      return timeDiff
    }

    if (left.kind === 'trace' && right.kind === 'trace') {
      return left.entry.sequence - right.entry.sequence
    }

    return (left.kind === 'step' ? left.entry.id : left.entry.id).localeCompare(
      right.kind === 'step' ? right.entry.id : right.entry.id,
    )
  })

  return items
}

function mergeTextBlock(
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

function seedRunBlocks(run: TicketRun): TicketRunTranscriptBlock[] {
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

function buildLifecycleBlock(lifecycle: TicketRunLifecycleEvent): TicketRunTranscriptBlock | null {
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

function finalizeTerminalRunBlocks(state: TicketRunTranscriptState): TicketRunTranscriptState {
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

function mergeRun(runs: TicketRun[], run: TicketRun): TicketRun[] {
  const nextRuns = runs.filter((item) => item.id !== run.id)
  nextRuns.push(run)
  return sortRuns(nextRuns)
}

function sortRuns(runs: TicketRun[]): TicketRun[] {
  return runs
    .slice()
    .sort((left, right) =>
      right.attemptNumber !== left.attemptNumber
        ? right.attemptNumber - left.attemptNumber
        : Date.parse(right.createdAt) - Date.parse(left.createdAt),
    )
}

function shouldSwitchToRun(currentRun: TicketRun | null, incomingRun: TicketRun): boolean {
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

function isTerminalRunStatus(status: TicketRun['status'] | undefined): boolean {
  return status === 'completed' || status === 'failed' || status === 'stalled'
}

function readPayloadString(payload: Record<string, unknown>, key: string): string {
  const value = payload[key]
  return typeof value === 'string' ? value : ''
}

function hasBlock(blocks: TicketRunTranscriptBlock[], id: string): boolean {
  return blocks.some((block) => block.id === id)
}
