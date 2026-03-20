import type { Agent, AgentStatus, ActivityEvent, StreamEnvelope } from './types'

export function parseStreamEnvelope(frame: { data: string }): StreamEnvelope | null {
  try {
    return JSON.parse(frame.data) as StreamEnvelope
  } catch {
    return null
  }
}

export function parseAgentPatch(raw: unknown): (Partial<Agent> & { id: string }) | null {
  const source = unwrapObject(raw, 'agent')
  if (!source) {
    return null
  }

  const id = readString(source, 'id') ?? readString(source, 'agent_id')
  if (!id) {
    return null
  }

  const status = readString(source, 'status')
  return compactAgentPatch({
    id,
    provider_id: readString(source, 'provider_id'),
    project_id: readString(source, 'project_id'),
    name: readString(source, 'name'),
    status: isAgentStatus(status) ? status : undefined,
    current_ticket_id: readNullableString(source, 'current_ticket_id'),
    session_id: readString(source, 'session_id'),
    workspace_path: readString(source, 'workspace_path'),
    capabilities: readStringArray(source, 'capabilities'),
    total_tokens_used: readNumber(source, 'total_tokens_used'),
    total_tickets_completed: readNumber(source, 'total_tickets_completed'),
    last_heartbeat_at: readNullableString(source, 'last_heartbeat_at'),
  })
}

export function parseActivityEvent(raw: unknown, fallbackCreatedAt: string): ActivityEvent | null {
  const source = unwrapObject(raw, 'event')
  if (!source) {
    return null
  }

  const id = readString(source, 'id')
  const projectId = readString(source, 'project_id')
  const eventType = readString(source, 'event_type') ?? readString(source, 'type')
  if (!id || !projectId || !eventType) {
    return null
  }

  return {
    id,
    project_id: projectId,
    ticket_id: readNullableString(source, 'ticket_id') ?? null,
    agent_id: readNullableString(source, 'agent_id') ?? null,
    event_type: eventType,
    message: readString(source, 'message') ?? '',
    metadata: readRecord(source, 'metadata') ?? {},
    created_at: readString(source, 'created_at') ?? fallbackCreatedAt,
  }
}

function compactAgentPatch(patch: Partial<Agent> & { id: string }) {
  return Object.fromEntries(
    Object.entries(patch).filter(([, value]) => value !== undefined),
  ) as Partial<Agent> & { id: string }
}

function unwrapObject(raw: unknown, nestedKey: string) {
  if (!isRecord(raw)) {
    return null
  }

  const nested = raw[nestedKey]
  if (isRecord(nested)) {
    return nested
  }

  return raw
}

function readString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function readNullableString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  if (value === null) {
    return null
  }
  return typeof value === 'string' ? value : undefined
}

function readNumber(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'number' ? value : undefined
}

function readStringArray(source: Record<string, unknown>, key: string) {
  const value = source[key]
  if (!Array.isArray(value) || !value.every((item) => typeof item === 'string')) {
    return undefined
  }

  return [...value]
}

function readRecord(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return isRecord(value) ? value : undefined
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isAgentStatus(value?: string): value is AgentStatus {
  return (
    value === 'idle' ||
    value === 'claimed' ||
    value === 'running' ||
    value === 'failed' ||
    value === 'terminated'
  )
}
