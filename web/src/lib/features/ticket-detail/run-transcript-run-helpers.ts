import type { TicketRun, TicketRunStepEntry, TicketRunTranscriptState } from './types'
import { hasBlock, seedRunBlocks } from './run-transcript-blocks'

export function mergeHydratedRunBlocks(
  existingBlocks: TicketRunTranscriptState['blocks'],
  run: TicketRun,
): TicketRunTranscriptState['blocks'] {
  const nextBlocks = seedRunBlocks(run)
  for (const block of existingBlocks) {
    if (!hasBlock(nextBlocks, block.id)) {
      nextBlocks.push(block)
    }
  }
  return nextBlocks
}

export function insertBlockChronologically(
  blocks: TicketRunTranscriptState['blocks'],
  nextBlock: TicketRunTranscriptState['blocks'][number],
): TicketRunTranscriptState['blocks'] {
  const nextAt = readBlockTimestamp(nextBlock)
  if (!nextAt) {
    return [...blocks, nextBlock]
  }

  const nextTime = Date.parse(nextAt)
  for (let index = 0; index < blocks.length; index += 1) {
    const candidateAt = readBlockTimestamp(blocks[index])
    if (!candidateAt) {
      return [...blocks.slice(0, index), nextBlock, ...blocks.slice(index)]
    }
    if (Date.parse(candidateAt) > nextTime) {
      return [...blocks.slice(0, index), nextBlock, ...blocks.slice(index)]
    }
  }

  return [...blocks, nextBlock]
}

export function mergeHydratedRunSnapshot(
  existingRun: TicketRun | undefined,
  hydratedRun: TicketRun,
): TicketRun {
  if (!existingRun || existingRun.id !== hydratedRun.id) {
    return hydratedRun
  }

  if (isTerminalStatus(existingRun.status) && !isTerminalStatus(hydratedRun.status)) {
    return existingRun
  }

  if (isTerminalStatus(hydratedRun.status) && !isTerminalStatus(existingRun.status)) {
    return hydratedRun
  }

  return mergeDefinedRunFields(hydratedRun, existingRun)
}

export function mergeStreamingRunSnapshot(
  existingRun: TicketRun | undefined,
  incomingRun: TicketRun,
): TicketRun {
  if (!existingRun || existingRun.id !== incomingRun.id) {
    return incomingRun
  }

  if (isTerminalStatus(existingRun.status) && !isTerminalStatus(incomingRun.status)) {
    return existingRun
  }

  return mergeDefinedRunFields(existingRun, incomingRun)
}

export function mergeDefinedRunFields(
  baseRun: TicketRun,
  incomingRun: TicketRun | undefined,
): TicketRun {
  if (!incomingRun) {
    return baseRun
  }

  const definedEntries = Object.entries(incomingRun).filter(([, value]) => value !== undefined)
  return {
    ...baseRun,
    ...Object.fromEntries(definedEntries),
  } as TicketRun
}

export function mergeRunStepSnapshot(run: TicketRun, entry: TicketRunStepEntry): TicketRun {
  if (
    !entry.sourceTraceEventId &&
    run.currentStepStatus &&
    run.lastHeartbeatAt &&
    Date.parse(entry.createdAt) < Date.parse(run.lastHeartbeatAt)
  ) {
    return run
  }

  return {
    ...run,
    currentStepStatus: entry.stepStatus,
    currentStepSummary: entry.summary || run.currentStepSummary,
  }
}

function readBlockTimestamp(block: TicketRunTranscriptState['blocks'][number]): string | undefined {
  switch (block.kind) {
    case 'phase':
    case 'step':
    case 'tool_call':
    case 'task_status':
    case 'diff':
    case 'interrupt':
      return block.at
    default:
      return undefined
  }
}

function isTerminalStatus(status: TicketRun['status']): boolean {
  return (
    status === 'completed' || status === 'failed' || status === 'interrupted' || status === 'ended'
  )
}
