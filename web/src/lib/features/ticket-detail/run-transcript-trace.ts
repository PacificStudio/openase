import {
  asRecord,
  buildProviderStateDetail,
  buildReasoningDetail,
  buildTaskDetail,
  readBoolean,
  parseUnifiedDiffPayloads,
  readString,
} from '$lib/features/chat'
import type {
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptState,
} from './types'
import { hasBlock, mergeRunTextBlock } from './run-transcript-blocks'
import { buildInterruptBlock } from './run-transcript-interrupts'
import { cacheSelectedState } from './run-transcript-selection'

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
      return cacheSelectedState(mergeRunTextBlock(state, entry, 'assistant_message'))
    case 'command_output_delta':
    case 'command_output_snapshot':
      return cacheSelectedState(mergeRunTextBlock(state, entry, 'terminal_output'))
    case 'tool_call_started':
      return appendTraceBlocks(state, [
        {
          kind: 'tool_call',
          id: `tool:${entry.id}`,
          toolName: readPayloadString(entry.payload, 'tool') || entry.output || entry.stream,
          arguments: entry.payload.arguments,
          summary: readPayloadString(entry.payload, 'phase') || undefined,
          at: entry.createdAt,
        },
      ])
    case 'thread_status':
      return appendTraceBlocks(state, [
        {
          kind: 'task_status',
          id: `status:${entry.id}`,
          statusType: 'thread_status',
          title: 'Codex thread status',
          detail: buildProviderStateDetail(asRecord(entry.payload)) || entry.output || undefined,
          raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
          at: entry.createdAt,
        },
      ])
    case 'reasoning_updated':
      return appendTraceBlocks(state, [
        {
          kind: 'task_status',
          id: `reasoning:${entry.id}`,
          statusType: 'reasoning_updated',
          title: 'Reasoning update',
          detail: entry.output || buildReasoningDetail(asRecord(entry.payload)) || undefined,
          raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
          at: entry.createdAt,
        },
      ])
    case 'task_started':
      return appendTraceBlocks(state, [buildTaskStatusBlock(entry, 'task_started', 'Task started')])
    case 'task_progress': {
      const payload = asRecord(entry.payload)
      const stream = readString(payload, 'stream')
      const content = readString(payload, 'text') || entry.output
      if (stream === 'command' && content) {
        return cacheSelectedState(
          mergeRunTextBlock(
            state,
            {
              ...entry,
              kind: readBoolean(payload, 'snapshot')
                ? 'command_output_snapshot'
                : 'command_output_delta',
              stream,
              output: content,
              payload: entry.payload,
            },
            'terminal_output',
          ),
        )
      }
      return appendTraceBlocks(state, [
        buildTaskStatusBlock(entry, 'task_progress', 'Task progress'),
      ])
    }
    case 'task_notification': {
      const payload = asRecord(entry.payload)
      const tool = readString(payload, 'tool')
      if (tool) {
        return appendTraceBlocks(state, [
          {
            kind: 'tool_call',
            id: `tool:${entry.id}`,
            toolName: tool,
            arguments: payload?.arguments,
            summary: undefined,
            at: entry.createdAt,
          },
        ])
      }
      return appendTraceBlocks(state, [
        buildTaskStatusBlock(entry, 'task_notification', 'Task notification'),
      ])
    }
    case 'session_state':
      return appendTraceBlocks(state, [
        {
          kind: 'task_status',
          id: `status:${entry.id}`,
          statusType: 'session_state',
          title: 'Claude session status',
          detail: buildProviderStateDetail(asRecord(entry.payload)) || entry.output || undefined,
          raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
          at: entry.createdAt,
        },
      ])
    case 'error':
      return appendTraceBlocks(state, [buildTaskStatusBlock(entry, 'error', 'Turn failed')])
    case 'turn_diff_updated':
      return appendTraceBlocks(
        state,
        createDiffBlocksFromTraceEntry(entry).map((block, index) => ({
          ...block,
          id: index === 0 ? `diff:${entry.id}` : `diff:${entry.id}:${index + 1}`,
        })),
      )
    case 'approval_requested':
    case 'user_input_requested': {
      const interruptBlock = buildInterruptBlock(entry)
      if (!interruptBlock || hasBlock(state.blocks, interruptBlock.id)) {
        return state
      }
      return cacheSelectedState({
        ...state,
        blocks: [...state.blocks, interruptBlock],
      })
    }
    default:
      return state
  }
}

function buildTaskStatusBlock(
  entry: TicketRunTraceEntry,
  statusType: Extract<
    Extract<TicketRunTranscriptBlock, { kind: 'task_status' }>['statusType'],
    'task_started' | 'task_progress' | 'task_notification' | 'error'
  >,
  title: string,
): Extract<TicketRunTranscriptBlock, { kind: 'task_status' }> {
  return {
    kind: 'task_status',
    id: `status:${entry.id}`,
    statusType,
    title,
    detail: buildTaskDetail(asRecord(entry.payload)) || entry.output || undefined,
    raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
    at: entry.createdAt,
  }
}

function appendTraceBlocks(
  state: TicketRunTranscriptState,
  blocks: TicketRunTranscriptBlock[],
): TicketRunTranscriptState {
  const nextBlocks = blocks.filter((block) => !hasBlock(state.blocks, block.id))
  if (nextBlocks.length === 0) {
    return state
  }

  return cacheSelectedState({
    ...state,
    blocks: [...state.blocks, ...nextBlocks],
  })
}

function createDiffBlocksFromTraceEntry(
  entry: TicketRunTraceEntry,
): Array<Extract<TicketRunTranscriptBlock, { kind: 'diff' }>> {
  const diffText = readString(asRecord(entry.payload), 'diff') || entry.output
  if (!diffText) {
    return []
  }

  return parseUnifiedDiffPayloads(diffText).map((diff) => ({
    kind: 'diff',
    id: '',
    at: entry.createdAt,
    diff: {
      ...diff,
      entryId: entry.id,
    },
  }))
}

function readPayloadString(payload: Record<string, unknown>, key: string): string | null {
  return readString(asRecord(payload), key) ?? null
}
