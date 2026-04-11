import type {
  TicketRun,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptPage,
  TicketRunTranscriptState,
} from './types'
import {
  buildRunTimeline,
  finalizeTerminalRunBlocks,
  hasBlock,
  mergeRun,
  seedRunBlocks,
} from './run-transcript-blocks'
import { applyTicketRunTraceEntry as reduceTicketRunTraceEntry } from './run-transcript-trace'
import { syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'
import {
  insertBlockChronologically,
  mergeDefinedRunFields,
  mergeRunStepSnapshot,
} from './run-transcript-run-helpers'
import {
  maxTicketRunEventCursor,
  buildTicketRunStepCursor,
  buildTicketRunTraceCursor,
  maxTicketRunTranscriptCursor,
} from './run-transcript-cursor'
import { createEmptyTicketRunTranscriptState } from './run-transcript-state'

export function mergeTranscriptPageIntoState(
  state: TicketRunTranscriptState,
  runId: string,
  page: TicketRunTranscriptPage,
): TicketRunTranscriptState {
  let nextState = state
  for (const item of page.items) {
    nextState =
      item.kind === 'step'
        ? mergeStepEntryIntoState(nextState, item.stepEntry)
        : mergeTraceEntryIntoState(nextState, item.traceEntry)
  }

  return {
    ...nextState,
    pageInfoByRun: {
      ...nextState.pageInfoByRun,
      [runId]: {
        hasOlder: page.hasOlder,
        hiddenOlderCount: page.hiddenOlderCount,
        oldestCursor: page.oldestCursor ?? nextState.pageInfoByRun[runId]?.oldestCursor,
        newestCursor: maxTicketRunTranscriptCursor(
          nextState.pageInfoByRun[runId]?.newestCursor,
          page.newestCursor,
        ),
        oldestEventCursor:
          page.oldestEventCursor ?? nextState.pageInfoByRun[runId]?.oldestEventCursor,
        newestEventCursor: maxTicketRunEventCursor(
          nextState.pageInfoByRun[runId]?.newestEventCursor,
          page.newestEventCursor,
        ),
      },
    },
  }
}

export function mergeStepEntryIntoState(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  const stepEntries = state.stepEntriesByRun[entry.agentRunId] ?? []
  if (stepEntries.some((item) => isEquivalentStepEntry(item, entry))) {
    return state
  }

  return {
    ...state,
    stepEntriesByRun: {
      ...state.stepEntriesByRun,
      [entry.agentRunId]: sortStepEntries([...stepEntries, entry]),
    },
  }
}

export function mergeTraceEntryIntoState(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
): TicketRunTranscriptState {
  const traceEntries = state.traceEntriesByRun[entry.agentRunId] ?? []
  if (traceEntries.some((item) => isEquivalentTraceEntry(item, entry))) {
    return state
  }

  return {
    ...state,
    traceEntriesByRun: {
      ...state.traceEntriesByRun,
      [entry.agentRunId]: sortTraceEntries([...traceEntries, entry]),
    },
  }
}

export function rebuildRunTranscriptState(
  state: TicketRunTranscriptState,
  runId: string,
  baseRun: TicketRun,
  newestCursor?: string,
): TicketRunTranscriptState {
  const traceEntries = state.traceEntriesByRun[runId] ?? []
  const stepEntries = state.stepEntriesByRun[runId] ?? []
  const lifecycleBlocks = state.lifecycleBlocksByRun[runId] ?? []
  const timeline = buildRunTimeline(stepEntries, traceEntries)
  const replayBaseRun: TicketRun = {
    ...baseRun,
    currentStepStatus: undefined,
    currentStepSummary: undefined,
    lastHeartbeatAt: undefined,
  }

  let replayState: TicketRunTranscriptState = {
    ...createEmptyTicketRunTranscriptState(),
    runs: [replayBaseRun],
    selectedRunId: runId,
    followLatest: true,
    currentRun: replayBaseRun,
    blocks: mergeSeedAndLifecycleBlocks(replayBaseRun, lifecycleBlocks),
  }

  for (const item of timeline) {
    replayState =
      item.kind === 'step'
        ? reduceTicketRunStepEntryForCurrentRun(replayState, item.entry)
        : reduceTicketRunTraceEntry(replayState, item.entry)
  }
  replayState = finalizeTerminalRunBlocks(replayState)

  const oldestCursor = state.pageInfoByRun[runId]?.oldestCursor ?? readTimelineCursor(timeline[0])
  const latestTimelineCursor = readTimelineCursor(timeline.at(-1))
  const rebuiltRun = mergeDefinedRunFields(baseRun, replayState.currentRun ?? undefined)

  return syncSelectedRun({
    ...state,
    runs: mergeRun(state.runs, rebuiltRun),
    blockCache: {
      ...syncSelectedBlocks(state),
      [runId]: replayState.blocks,
    },
    pageInfoByRun: {
      ...state.pageInfoByRun,
      [runId]: {
        hasOlder: state.pageInfoByRun[runId]?.hasOlder ?? false,
        hiddenOlderCount: state.pageInfoByRun[runId]?.hiddenOlderCount ?? 0,
        oldestCursor,
        newestCursor: maxTicketRunTranscriptCursor(
          maxTicketRunTranscriptCursor(
            state.pageInfoByRun[runId]?.newestCursor,
            latestTimelineCursor,
          ),
          newestCursor,
        ),
        oldestEventCursor: state.pageInfoByRun[runId]?.oldestEventCursor,
        newestEventCursor: state.pageInfoByRun[runId]?.newestEventCursor,
      },
    },
  })
}

function reduceTicketRunStepEntryForCurrentRun(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  const runs = state.runs.map((run) =>
    run.id === entry.agentRunId ? mergeRunStepSnapshot(run, entry) : run,
  )

  const currentRun = state.currentRun
  if (!currentRun || currentRun.id !== entry.agentRunId) {
    return {
      ...state,
      runs,
    }
  }

  const nextCurrentRun = mergeRunStepSnapshot(currentRun, entry)
  if (entry.sourceTraceEventId || hasBlock(state.blocks, `step:${entry.id}`)) {
    return {
      ...state,
      runs,
      currentRun: nextCurrentRun,
    }
  }

  return {
    ...state,
    runs,
    currentRun: nextCurrentRun,
    blocks: [
      ...state.blocks,
      {
        kind: 'step' as const,
        id: `step:${entry.id}`,
        stepStatus: entry.stepStatus,
        summary: entry.summary,
        at: entry.createdAt,
      },
    ],
  }
}

function mergeSeedAndLifecycleBlocks(
  run: TicketRun,
  lifecycleBlocks: TicketRunTranscriptBlock[],
): TicketRunTranscriptBlock[] {
  let blocks = seedRunBlocks(run)
  for (const block of lifecycleBlocks) {
    if (!hasBlock(blocks, block.id)) {
      blocks = insertBlockChronologically(blocks, block)
    }
  }
  return blocks
}

function sortTraceEntries(entries: TicketRunTraceEntry[]): TicketRunTraceEntry[] {
  return entries
    .slice()
    .sort((left, right) =>
      left.sequence !== right.sequence
        ? left.sequence - right.sequence
        : left.id.localeCompare(right.id),
    )
}

function sortStepEntries(entries: TicketRunStepEntry[]): TicketRunStepEntry[] {
  return entries
    .slice()
    .sort((left, right) =>
      Date.parse(left.createdAt) !== Date.parse(right.createdAt)
        ? Date.parse(left.createdAt) - Date.parse(right.createdAt)
        : left.id.localeCompare(right.id),
    )
}

function readTimelineCursor(
  item: ReturnType<typeof buildRunTimeline>[number] | undefined,
): string | undefined {
  if (!item) {
    return undefined
  }
  return item.kind === 'step'
    ? buildTicketRunStepCursor(item.entry)
    : buildTicketRunTraceCursor(item.entry)
}

function isEquivalentStepEntry(left: TicketRunStepEntry, right: TicketRunStepEntry): boolean {
  return (
    left.id === right.id ||
    (left.agentRunId === right.agentRunId &&
      left.createdAt === right.createdAt &&
      left.stepStatus === right.stepStatus &&
      left.summary === right.summary &&
      left.sourceTraceEventId === right.sourceTraceEventId)
  )
}

function isEquivalentTraceEntry(left: TicketRunTraceEntry, right: TicketRunTraceEntry): boolean {
  return (
    left.id === right.id ||
    (left.agentRunId === right.agentRunId &&
      left.sequence === right.sequence &&
      left.kind === right.kind &&
      left.stream === right.stream &&
      left.createdAt === right.createdAt &&
      left.output === right.output &&
      JSON.stringify(left.payload) === JSON.stringify(right.payload))
  )
}
