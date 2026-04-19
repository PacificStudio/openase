import { ApiError } from '$lib/api/client'

export function formatProviderDeleteError(caughtError: unknown) {
  if (!(caughtError instanceof ApiError)) {
    return 'Failed to delete provider.'
  }
  if (caughtError.code !== 'PROVIDER_IN_USE') {
    return caughtError.detail
  }

  const details = caughtError.details
  if (!details || typeof details !== 'object') {
    return caughtError.detail
  }

  const record = details as Record<string, unknown>
  const parts: string[] = []
  if (record.organization_default === true) {
    parts.push('organization default')
  }
  const projectDefaults = namedReferences(record.project_defaults, 'project defaults')
  if (projectDefaults) parts.push(projectDefaults)
  const agents = namedReferences(record.agents, 'agents', 'name')
  if (agents) parts.push(agents)
  const conversations = namedReferences(record.chat_conversations, 'chat conversations')
  if (conversations) parts.push(conversations)
  const principals = namedReferences(
    record.conversation_principals,
    'project conversation principals',
    'name',
  )
  if (principals) parts.push(principals)
  const runs = countReferences(record.conversation_runs, 'project conversation runs')
  if (runs) parts.push(runs)
  const agentRuns = countReferences(record.agent_runs, 'agent runs')
  if (agentRuns) parts.push(agentRuns)

  if (parts.length === 0) {
    return caughtError.detail
  }

  return `Provider is still in use by ${parts.join(', ')}.`
}

function namedReferences(value: unknown, label: string, nameField: string = 'name') {
  if (!Array.isArray(value) || value.length === 0) {
    return ''
  }
  const names = value
    .map((item) =>
      item &&
      typeof item === 'object' &&
      typeof (item as Record<string, unknown>)[nameField] === 'string'
        ? ((item as Record<string, unknown>)[nameField] as string)
        : '',
    )
    .filter(Boolean)
  if (names.length === 0) {
    return `${value.length} ${label}`
  }
  return `${label}: ${names.join(', ')}`
}

function countReferences(value: unknown, label: string) {
  if (!Array.isArray(value) || value.length === 0) {
    return ''
  }
  return `${value.length} ${label}`
}
