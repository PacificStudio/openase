import type { ChatDiffPayload } from '$lib/api/chat'

export type TicketRunTranscriptInterruptOption = {
  id: string
  label: string
  rawDecision?: string
}

export type TicketRunTranscriptBlock =
  | { kind: 'phase'; id: string; phase: string; at: string; summary: string }
  | { kind: 'step'; id: string; stepStatus: string; summary: string; at: string }
  | { kind: 'assistant_message'; id: string; itemId?: string; text: string; streaming: boolean }
  | {
      kind: 'tool_call'
      id: string
      toolName: string
      arguments?: unknown
      summary?: string
      at: string
    }
  | {
      kind: 'terminal_output'
      id: string
      itemId?: string
      stream: string
      command?: string
      phase?: string
      text: string
      streaming: boolean
    }
  | {
      kind: 'task_status'
      id: string
      statusType:
        | 'task_started'
        | 'task_progress'
        | 'task_notification'
        | 'thread_status'
        | 'reasoning_updated'
        | 'session_state'
        | 'error'
      title: string
      detail?: string
      raw?: Record<string, unknown>
      at: string
    }
  | { kind: 'diff'; id: string; at: string; diff: ChatDiffPayload }
  | {
      kind: 'interrupt'
      id: string
      interruptKind: string
      title: string
      summary: string
      at: string
      payload: Record<string, unknown>
      options: TicketRunTranscriptInterruptOption[]
    }
  | { kind: 'result'; id: string; outcome: 'completed' | 'failed' | 'ended'; summary: string }

export type TicketRunTranscriptState = {
  runs: import('./types').TicketRun[]
  selectedRunId: string | null
  followLatest: boolean
  currentRun: import('./types').TicketRun | null
  blocks: TicketRunTranscriptBlock[]
  blockCache: Record<string, TicketRunTranscriptBlock[]>
}
