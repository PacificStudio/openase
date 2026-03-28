import {
  addTicketDependency,
  addTicketExternalLink,
  createTicket,
  createTicketComment,
  createWorkflow,
  updateProject,
  updateTicket,
  updateTicketComment,
  updateWorkflow,
} from '$lib/api/openase'

export function parseCreateTicketBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof createTicket>[1] {
  const object = requireBodyObject(body)
  return {
    title: readRequiredString(object, 'title'),
    description: readOptionalString(object, 'description'),
    status_id: readOptionalNullableString(object, 'status_id'),
    priority: readOptionalNullableString(object, 'priority'),
    type: readOptionalNullableString(object, 'type'),
    workflow_id: readOptionalNullableString(object, 'workflow_id'),
    created_by: readOptionalNullableString(object, 'created_by'),
    parent_ticket_id: readOptionalNullableString(object, 'parent_ticket_id'),
    external_ref: readOptionalNullableString(object, 'external_ref'),
    budget_usd: readOptionalNullableNumber(object, 'budget_usd'),
  }
}

export function parseUpdateTicketBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof updateTicket>[1] {
  const object = requireBodyObject(body)
  return {
    title: readOptionalNullableString(object, 'title'),
    description: readOptionalNullableString(object, 'description'),
    status_id: readOptionalNullableString(object, 'status_id'),
    priority: readOptionalNullableString(object, 'priority'),
    type: readOptionalNullableString(object, 'type'),
    workflow_id: readOptionalNullableString(object, 'workflow_id'),
    created_by: readOptionalNullableString(object, 'created_by'),
    parent_ticket_id: readOptionalNullableString(object, 'parent_ticket_id'),
    external_ref: readOptionalNullableString(object, 'external_ref'),
    budget_usd: readOptionalNullableNumber(object, 'budget_usd'),
  }
}

export function parseCreateCommentBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof createTicketComment>[1] {
  const object = requireBodyObject(body)
  return {
    body: readRequiredString(object, 'body'),
    created_by: readOptionalNullableString(object, 'created_by'),
  }
}

export function parseUpdateCommentBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof updateTicketComment>[2] {
  const object = requireBodyObject(body)
  return {
    body: readRequiredString(object, 'body'),
  }
}

export function parseDependencyBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof addTicketDependency>[1] {
  const object = requireBodyObject(body)
  return {
    target_ticket_id: readRequiredString(object, 'target_ticket_id'),
    type: readRequiredString(object, 'type'),
  }
}

export function parseExternalLinkBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof addTicketExternalLink>[1] {
  const object = requireBodyObject(body)
  return {
    type: readRequiredString(object, 'type'),
    url: readRequiredString(object, 'url'),
    external_id: readRequiredString(object, 'external_id'),
    title: readOptionalNullableString(object, 'title'),
    status: readOptionalNullableString(object, 'status'),
    relation: readOptionalNullableString(object, 'relation'),
  }
}

export function parseProjectUpdateBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof updateProject>[1] {
  const object = requireBodyObject(body)
  return {
    default_agent_provider_id: readOptionalNullableString(object, 'default_agent_provider_id'),
    default_workflow_id: readOptionalNullableString(object, 'default_workflow_id'),
    description: readOptionalNullableString(object, 'description'),
    max_concurrent_agents: readOptionalNullableNumber(object, 'max_concurrent_agents'),
    name: readOptionalNullableString(object, 'name'),
    slug: readOptionalNullableString(object, 'slug'),
    status: readOptionalNullableString(object, 'status'),
  }
}

export function parseCreateWorkflowBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof createWorkflow>[1] {
  const object = requireBodyObject(body)
  return {
    agent_id: readRequiredString(object, 'agent_id'),
    finish_status_ids: readRequiredStringArray(object, 'finish_status_ids'),
    harness_content: readOptionalString(object, 'harness_content'),
    harness_path: readOptionalNullableString(object, 'harness_path'),
    hooks: readOptionalObject(object, 'hooks'),
    is_active: readOptionalNullableBoolean(object, 'is_active'),
    max_concurrent: readOptionalNullableNumber(object, 'max_concurrent'),
    max_retry_attempts: readOptionalNullableNumber(object, 'max_retry_attempts'),
    name: readOptionalString(object, 'name'),
    pickup_status_ids: readRequiredStringArray(object, 'pickup_status_ids'),
    stall_timeout_minutes: readOptionalNullableNumber(object, 'stall_timeout_minutes'),
    timeout_minutes: readOptionalNullableNumber(object, 'timeout_minutes'),
    type: readOptionalString(object, 'type'),
  }
}

export function parseUpdateWorkflowBody(
  body: Record<string, unknown> | undefined,
): Parameters<typeof updateWorkflow>[1] {
  const object = requireBodyObject(body)
  return {
    agent_id: readOptionalNullableString(object, 'agent_id'),
    finish_status_ids: readOptionalStringArray(object, 'finish_status_ids'),
    harness_path: readOptionalNullableString(object, 'harness_path'),
    hooks: readOptionalNullableObject(object, 'hooks'),
    is_active: readOptionalNullableBoolean(object, 'is_active'),
    max_concurrent: readOptionalNullableNumber(object, 'max_concurrent'),
    max_retry_attempts: readOptionalNullableNumber(object, 'max_retry_attempts'),
    name: readOptionalNullableString(object, 'name'),
    pickup_status_ids: readOptionalStringArray(object, 'pickup_status_ids'),
    stall_timeout_minutes: readOptionalNullableNumber(object, 'stall_timeout_minutes'),
    timeout_minutes: readOptionalNullableNumber(object, 'timeout_minutes'),
    type: readOptionalNullableString(object, 'type'),
  }
}

export function parseHarnessBody(body: Record<string, unknown> | undefined) {
  const object = requireBodyObject(body)
  return readRequiredString(object, 'content')
}

export function parseSkillsBody(body: Record<string, unknown> | undefined) {
  const object = requireBodyObject(body)
  return readRequiredStringArray(object, 'skills')
}

function requireBodyObject(body: Record<string, unknown> | undefined) {
  if (!body) {
    throw new Error('Proposed action body is required.')
  }

  return body
}

function readRequiredString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Proposed action field ${key} must be a non-empty string.`)
  }
  return value
}

function readOptionalString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Proposed action field ${key} must be a non-empty string.`)
  }
  return value
}

function readOptionalNullableString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Proposed action field ${key} must be a non-empty string or null.`)
  }
  return value
}

function readOptionalNullableNumber(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Proposed action field ${key} must be a number or null.`)
  }
  return value
}

function readOptionalNullableBoolean(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'boolean') {
    throw new Error(`Proposed action field ${key} must be a boolean or null.`)
  }
  return value
}

function readRequiredStringArray(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (
    !Array.isArray(value) ||
    value.some((item) => typeof item !== 'string' || item.trim() === '')
  ) {
    throw new Error(`Proposed action field ${key} must be an array of non-empty strings.`)
  }
  return [...value]
}

function readOptionalStringArray(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined) {
    return undefined
  }
  if (
    !Array.isArray(value) ||
    value.some((item) => typeof item !== 'string' || item.trim() === '')
  ) {
    throw new Error(`Proposed action field ${key} must be an array of non-empty strings.`)
  }
  return [...value]
}

function readOptionalNullableObject(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Proposed action field ${key} must be an object or null.`)
  }
  return { ...(value as Record<string, unknown>) }
}

function readOptionalObject(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (value === undefined) {
    return undefined
  }
  if (value == null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Proposed action field ${key} must be an object.`)
  }
  return { ...(value as Record<string, unknown>) }
}
