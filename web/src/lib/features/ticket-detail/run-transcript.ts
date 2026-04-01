import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptState,
} from './types'
import {
  buildLifecycleBlock,
  buildRunTimeline,
  finalizeTerminalRunBlocks,
  hasBlock,
  mergeRun,
  mergeRunTextBlock,
  readPayloadString,
  seedRunBlocks,
  shouldSwitchToRun,
  sortTicketRuns,
} from './run-transcript-blocks'

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
  const nextRuns = sortTicketRuns(runs)
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

  const timeline = buildRunTimeline(detail.stepEntries, detail.traceEntries)
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
      return mergeRunTextBlock(state, entry, 'assistant_message')
    case 'command_output_delta':
    case 'command_output_snapshot':
      return mergeRunTextBlock(state, entry, 'terminal_output')
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
