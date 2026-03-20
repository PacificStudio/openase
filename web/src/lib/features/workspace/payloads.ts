import type {
  ActivityEvent,
  ActivityPayload,
  Agent,
  AgentPayload,
  HRAdvisorPayload,
  HRAdvisorRecommendation,
  StatusPayload,
  Ticket,
  TicketDependency,
  TicketPayload,
  TicketReference,
  TicketStatus,
  Workflow,
  WorkflowListPayload,
} from './types'
import {
  asRecord,
  parseArray,
  parseStringArray,
  parseUnknownRecord,
  readBoolean,
  readNullableString,
  readNumber,
  readString,
} from './payload-helpers'

export function parseAgentPayload(raw: unknown): AgentPayload {
  const source = asRecord(raw)
  return {
    agents: parseArray(source.agents, parseAgent),
  }
}

export function parseActivityPayload(raw: unknown): ActivityPayload {
  const source = asRecord(raw)
  return {
    events: parseArray(source.events, parseActivityEvent),
  }
}

export function parseStatusPayload(raw: unknown): StatusPayload {
  const source = asRecord(raw)
  return {
    statuses: parseArray(source.statuses, parseTicketStatus),
  }
}

export function parseTicketPayload(raw: unknown): TicketPayload {
  const source = asRecord(raw)
  return {
    tickets: parseArray(source.tickets, parseTicket),
  }
}

export function parseWorkflowListPayload(raw: unknown): WorkflowListPayload {
  const source = asRecord(raw)
  return {
    workflows: parseArray(source.workflows, parseWorkflow),
  }
}

export function parseHRAdvisorPayload(raw: unknown): HRAdvisorPayload {
  const source = asRecord(raw)
  return {
    project_id: readString(source, 'project_id'),
    summary: parseHRAdvisorSummary(source.summary),
    staffing: parseHRAdvisorStaffing(source.staffing),
    recommendations: parseArray(source.recommendations, parseHRAdvisorRecommendation),
  }
}

function parseAgent(raw: unknown): Agent | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    provider_id: readString(source, 'provider_id'),
    project_id: readString(source, 'project_id'),
    name: readString(source, 'name'),
    status: readString(source, 'status') || 'idle',
    current_ticket_id: readNullableString(source, 'current_ticket_id'),
    session_id: readString(source, 'session_id'),
    workspace_path: readString(source, 'workspace_path'),
    capabilities: parseStringArray(source.capabilities),
    total_tokens_used: readNumber(source, 'total_tokens_used'),
    total_tickets_completed: readNumber(source, 'total_tickets_completed'),
    last_heartbeat_at: readNullableString(source, 'last_heartbeat_at'),
  }
}

function parseActivityEvent(raw: unknown): ActivityEvent | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    project_id: readString(source, 'project_id'),
    ticket_id: readNullableString(source, 'ticket_id'),
    agent_id: readNullableString(source, 'agent_id'),
    event_type: readString(source, 'event_type') || 'unknown',
    message: readString(source, 'message'),
    metadata: parseUnknownRecord(source.metadata),
    created_at: readString(source, 'created_at') || '1970-01-01T00:00:00Z',
  }
}

function parseTicketStatus(raw: unknown): TicketStatus | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    project_id: readString(source, 'project_id'),
    name: readString(source, 'name'),
    color: readString(source, 'color'),
    icon: readString(source, 'icon'),
    position: readNumber(source, 'position'),
    is_default: readBoolean(source, 'is_default'),
    description: readString(source, 'description'),
  }
}

function parseTicket(raw: unknown): Ticket | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    project_id: readString(source, 'project_id'),
    identifier: readString(source, 'identifier'),
    title: readString(source, 'title'),
    description: readString(source, 'description'),
    status_id: readString(source, 'status_id'),
    status_name: readString(source, 'status_name'),
    priority: readString(source, 'priority') || 'medium',
    type: readString(source, 'type') || 'unknown',
    workflow_id: readNullableString(source, 'workflow_id'),
    created_by: readString(source, 'created_by'),
    parent: parseTicketReference(source.parent),
    children: parseArray(source.children, parseTicketReference),
    dependencies: parseArray(source.dependencies, parseTicketDependency),
    external_ref: readString(source, 'external_ref'),
    budget_usd: readNumber(source, 'budget_usd'),
    cost_amount: readNumber(source, 'cost_amount'),
    attempt_count: readNumber(source, 'attempt_count'),
    consecutive_errors: readNumber(source, 'consecutive_errors'),
    next_retry_at: readNullableString(source, 'next_retry_at'),
    retry_paused: readBoolean(source, 'retry_paused'),
    pause_reason: readString(source, 'pause_reason'),
    created_at: readString(source, 'created_at') || '1970-01-01T00:00:00Z',
  }
}

function parseTicketReference(raw: unknown): TicketReference | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    identifier: readString(source, 'identifier'),
    title: readString(source, 'title'),
    status_id: readString(source, 'status_id'),
    status_name: readString(source, 'status_name'),
  }
}

function parseTicketDependency(raw: unknown): TicketDependency | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  const target = parseTicketReference(source.target)
  if (!id || !target) {
    return null
  }

  return {
    id,
    type: readString(source, 'type') || 'unknown',
    target,
  }
}

function parseWorkflow(raw: unknown): Workflow | null {
  const source = asRecord(raw)
  const id = readString(source, 'id')
  if (!id) {
    return null
  }

  return {
    id,
    project_id: readString(source, 'project_id'),
    name: readString(source, 'name'),
    type: readString(source, 'type') || 'custom',
    harness_path: readString(source, 'harness_path'),
    harness_content: readNullableString(source, 'harness_content'),
    hooks: parseUnknownRecord(source.hooks),
    max_concurrent: readNumber(source, 'max_concurrent'),
    max_retry_attempts: readNumber(source, 'max_retry_attempts'),
    timeout_minutes: readNumber(source, 'timeout_minutes'),
    stall_timeout_minutes: readNumber(source, 'stall_timeout_minutes'),
    version: readNumber(source, 'version'),
    is_active: readBoolean(source, 'is_active'),
    pickup_status_id: readString(source, 'pickup_status_id'),
    finish_status_id: readNullableString(source, 'finish_status_id'),
  }
}

function parseHRAdvisorSummary(raw: unknown): HRAdvisorPayload['summary'] {
  const source = asRecord(raw)
  return {
    open_tickets: readNumber(source, 'open_tickets'),
    coding_tickets: readNumber(source, 'coding_tickets'),
    failing_tickets: readNumber(source, 'failing_tickets'),
    blocked_tickets: readNumber(source, 'blocked_tickets'),
    active_agents: readNumber(source, 'active_agents'),
    workflow_count: readNumber(source, 'workflow_count'),
    recent_activity_count: readNumber(source, 'recent_activity_count'),
    active_workflow_types: parseStringArray(source.active_workflow_types),
  }
}

function parseHRAdvisorStaffing(raw: unknown): HRAdvisorPayload['staffing'] {
  const source = asRecord(raw)
  return {
    developers: readNumber(source, 'developers'),
    qa: readNumber(source, 'qa'),
    docs: readNumber(source, 'docs'),
    security: readNumber(source, 'security'),
    product: readNumber(source, 'product'),
    research: readNumber(source, 'research'),
  }
}

function parseHRAdvisorRecommendation(raw: unknown): HRAdvisorRecommendation | null {
  const source = asRecord(raw)
  const roleSlug = readString(source, 'role_slug')
  if (!roleSlug) {
    return null
  }

  return {
    role_slug: roleSlug,
    role_name: readString(source, 'role_name'),
    workflow_type: readString(source, 'workflow_type') || 'custom',
    summary: readString(source, 'summary'),
    harness_path: readString(source, 'harness_path'),
    priority: readString(source, 'priority') || 'low',
    reason: readString(source, 'reason'),
    evidence: parseStringArray(source.evidence),
    suggested_headcount: readNumber(source, 'suggested_headcount'),
    suggested_workflow_name: readString(source, 'suggested_workflow_name'),
    activation_ready: readBoolean(source, 'activation_ready'),
    active_workflow_name: readNullableString(source, 'active_workflow_name'),
  }
}
