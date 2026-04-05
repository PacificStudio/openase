import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptPage,
  TicketRunTranscriptState,
} from './types'
import {
  buildLifecycleBlock,
  buildRunTimeline,
  finalizeTerminalRunBlocks,
  hasBlock,
  mergeRun,
  seedRunBlocks,
} from './run-transcript-blocks'
import { applyTicketRunTraceEntry as reduceTicketRunTraceEntry } from './run-transcript-trace'
import { syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'
export {
  createEmptyTicketRunTranscriptState,
  selectTicketRun,
  setTicketRunList,
} from './run-transcript-state'
import {
  insertBlockChronologically,
  mergeDefinedRunFields,
  mergeHydratedRunSnapshot,
  mergeRunStepSnapshot,
  mergeStreamingRunSnapshot,
} from './run-transcript-run-helpers'
import {
  mapTicketRun,
  mapTicketRunCompletionSummary,
  mapTicketRunStepEntry,
  mapTicketRunStreamLifecycleEvent,
  mapTicketRunTraceEntry,
} from './run-transcript-data'
import {
  buildTicketRunStepCursor,
  buildTicketRunTraceCursor,
  maxTicketRunTranscriptCursor,
} from './run-transcript-cursor'
import type { TicketRunRecord, TicketRunStepRecord, TicketRunTraceRecord } from '$lib/api/contracts'
import { createEmptyTicketRunTranscriptState } from './run-transcript-state'

export function hydrateTicketRunDetail(
  state: TicketRunTranscriptState,
  detail: TicketRunDetail,
  options: { select?: boolean } = {},
): TicketRunTranscriptState {
  const existingRun = state.runs.find((item) => item.id === detail.run.id)
  const hydratedRun = mergeHydratedRunSnapshot(existingRun, detail.run)

  let nextState = mergeTranscriptPageIntoState(
    {
      ...state,
      runs: mergeRun(state.runs, hydratedRun),
    },
    detail.run.id,
    detail.transcriptPage,
  )
  nextState = rebuildRunTranscriptState(nextState, detail.run.id, hydratedRun)

  const selection =
    options.select === false
      ? {
          selectedRunId: nextState.selectedRunId,
          followLatest: nextState.followLatest,
        }
      : {
          selectedRunId: hydratedRun.id,
          followLatest: nextState.runs[0]?.id === hydratedRun.id,
        }

  return syncSelectedRun({
    ...nextState,
    ...selection,
  })
}

export function applyTicketRunStreamFrame(
  state: TicketRunTranscriptState,
  frame: Pick<SSEFrame, 'event' | 'data'>,
): TicketRunTranscriptState {
  try {
    switch (frame.event) {
      case 'ticket.run.lifecycle': {
        const payload = JSON.parse(frame.data) as {
          run: TicketRunRecord
          lifecycle: {
            event_type?: string
            eventType?: string
            message?: string
            created_at?: string
            createdAt?: string
          }
        }
        return applyTicketRunLifecycleEvent(
          state,
          mapTicketRun(payload.run),
          mapTicketRunStreamLifecycleEvent(payload.lifecycle),
        )
      }
      case 'ticket.run.trace': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunTraceRecord }
        return applyTicketRunTraceEntry(state, mapTicketRunTraceEntry(payload.entry))
      }
      case 'ticket.run.step': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunStepRecord }
        return applyTicketRunStepEntry(state, mapTicketRunStepEntry(payload.entry))
      }
      case 'ticket.run.summary': {
        const payload = JSON.parse(frame.data) as {
          run_id?: string
          runId?: string
          run?: TicketRunRecord
          completion_summary?: {
            status?: string
            markdown?: string | null
            json?: Record<string, unknown> | null
            generated_at?: string | null
            error?: string | null
          } | null
        }
        return applyTicketRunSummaryEvent(
          state,
          payload.run_id ?? payload.runId ?? '',
          payload.run ? mapTicketRun(payload.run) : undefined,
          mapTicketRunCompletionSummary(payload.completion_summary),
        )
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
  const currentRunForUpdate = getRunForUpdate(state, run.id)
  const nextRun = mergeStreamingRunSnapshot(currentRunForUpdate ?? undefined, run)
  const nextBlock = buildLifecycleBlock(lifecycle)
  const lifecycleBlocks = state.lifecycleBlocksByRun[run.id] ?? []
  const nextLifecycleBlocks =
    !nextBlock || hasBlock(lifecycleBlocks, nextBlock.id)
      ? lifecycleBlocks
      : insertBlockChronologically(lifecycleBlocks, nextBlock)

  return rebuildRunTranscriptState(
    {
      ...state,
      runs: mergeRun(state.runs, nextRun),
      lifecycleBlocksByRun: {
        ...state.lifecycleBlocksByRun,
        [run.id]: nextLifecycleBlocks,
      },
    },
    run.id,
    nextRun,
  )
}

export function applyTicketRunStepEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  const baseRun = getRunForUpdate(state, entry.agentRunId)
  if (!baseRun) {
    return state
  }

  const stepEntries = state.stepEntriesByRun[entry.agentRunId] ?? []
  if (!stepEntries.some((item) => isEquivalentStepEntry(item, entry))) {
    state = {
      ...state,
      stepEntriesByRun: {
        ...state.stepEntriesByRun,
        [entry.agentRunId]: sortStepEntries([...stepEntries, entry]),
      },
    }
  }

  return rebuildRunTranscriptState(state, entry.agentRunId, baseRun, buildTicketRunStepCursor(entry))
}

export function applyTicketRunTraceEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
): TicketRunTranscriptState {
  const baseRun = getRunForUpdate(state, entry.agentRunId)
  if (!baseRun) {
    return state
  }

  const traceEntries = state.traceEntriesByRun[entry.agentRunId] ?? []
  if (!traceEntries.some((item) => isEquivalentTraceEntry(item, entry))) {
    state = {
      ...state,
      traceEntriesByRun: {
        ...state.traceEntriesByRun,
        [entry.agentRunId]: sortTraceEntries([...traceEntries, entry]),
      },
    }
  }

  return rebuildRunTranscriptState(state, entry.agentRunId, baseRun, buildTicketRunTraceCursor(entry))
}

export function applyTicketRunSummaryEvent(
  state: TicketRunTranscriptState,
  runID: string,
  run: TicketRun | undefined,
  completionSummary: TicketRun['completionSummary'],
): TicketRunTranscriptState {
  if (!runID || (!completionSummary && !run)) {
    return state
  }

  const existingRun = getRunForUpdate(state, runID)
  if (!existingRun) {
    return state
  }

  const nextRun = mergeDefinedRunFields(existingRun, run)
  nextRun.completionSummary =
    completionSummary ?? run?.completionSummary ?? existingRun.completionSummary

  return rebuildRunTranscriptState(
    {
      ...state,
      runs: mergeRun(state.runs, nextRun),
    },
    runID,
    nextRun,
  )
}

function mergeTranscriptPageIntoState(
  state: TicketRunTranscriptState,
  runId: string,
  page: TicketRunTranscriptPage,
): TicketRunTranscriptState {
  const stepEntries = state.stepEntriesByRun[runId] ?? []
  const traceEntries = state.traceEntriesByRun[runId] ?? []
  let nextStepEntries = stepEntries
  let nextTraceEntries = traceEntries

  for (const item of page.items) {
    if (item.kind === 'step') {
      if (!nextStepEntries.some((entry) => isEquivalentStepEntry(entry, item.stepEntry))) {
        nextStepEntries = sortStepEntries([...nextStepEntries, item.stepEntry])
      }
      continue
    }
    if (!nextTraceEntries.some((entry) => isEquivalentTraceEntry(entry, item.traceEntry))) {
      nextTraceEntries = sortTraceEntries([...nextTraceEntries, item.traceEntry])
    }
  }

  return {
    ...state,
    stepEntriesByRun: {
      ...state.stepEntriesByRun,
      [runId]: nextStepEntries,
    },
    traceEntriesByRun: {
      ...state.traceEntriesByRun,
      [runId]: nextTraceEntries,
    },
    pageInfoByRun: {
      ...state.pageInfoByRun,
      [runId]: {
        hasOlder: page.hasOlder,
        hiddenOlderCount: page.hiddenOlderCount,
        oldestCursor: page.oldestCursor ?? state.pageInfoByRun[runId]?.oldestCursor,
        newestCursor: maxTicketRunTranscriptCursor(
          state.pageInfoByRun[runId]?.newestCursor,
          page.newestCursor,
        ),
      },
    },
  }
}

function rebuildRunTranscriptState(
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
  const oldestCursor =
    state.pageInfoByRun[runId]?.oldestCursor ??
    readTimelineCursor(timeline[0])
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
          maxTicketRunTranscriptCursor(state.pageInfoByRun[runId]?.newestCursor, latestTimelineCursor),
          newestCursor,
        ),
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
  return entries.slice().sort((left, right) =>
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

function getRunForUpdate(state: TicketRunTranscriptState, runId: string): TicketRun | null {
  return state.runs.find((item) => item.id === runId) ?? (state.currentRun?.id === runId ? state.currentRun : null)
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
