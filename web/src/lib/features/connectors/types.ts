export const connectorTypes = ['github', 'gitlab', 'jira', 'inbound-webhook', 'custom'] as const
export const connectorStatuses = ['active', 'paused', 'error'] as const
export const syncDirections = ['pull_only', 'push_only', 'bidirectional'] as const
export type ConnectorType = (typeof connectorTypes)[number]
export type ConnectorStatus = (typeof connectorStatuses)[number]
export type SyncDirection = (typeof syncDirections)[number]
export type ConnectorFilters = {
  labels: string[]
  exclude_labels: string[]
  states: string[]
  authors: string[]
}
export type ConnectorConfig = {
  type: ConnectorType
  base_url: string
  auth_token: string
  project_ref: string
  poll_interval: string
  sync_direction: SyncDirection
  filters: ConnectorFilters
  status_mapping: Record<string, string>
  webhook_secret: string
  auto_workflow: string
}
export type ConnectorStats = {
  total_synced: number
  synced_24h: number
  failed_count: number
}
export type IssueConnector = {
  id: string
  project_id: string
  type: ConnectorType
  name: string
  status: ConnectorStatus
  config: ConnectorConfig
  last_sync_at?: string | null
  last_error: string
  stats: ConnectorStats
}
export type ConnectorInput = {
  type: ConnectorType
  name: string
  status: ConnectorStatus
  config: ConnectorConfig
}
export type ConnectorForm = {
  type: ConnectorType
  name: string
  status: ConnectorStatus
  base_url: string
  auth_token: string
  project_ref: string
  poll_interval: string
  sync_direction: SyncDirection
  labels: string
  exclude_labels: string
  states: string
  authors: string
  status_mapping: string
  webhook_secret: string
  auto_workflow: string
}

export function defaultConnectorForm(): ConnectorForm {
  return {
    type: 'github',
    name: '',
    status: 'active',
    base_url: 'https://api.github.com',
    auth_token: '',
    project_ref: '',
    poll_interval: '5m',
    sync_direction: 'bidirectional',
    labels: '',
    exclude_labels: '',
    states: 'open',
    authors: '',
    status_mapping: '',
    webhook_secret: '',
    auto_workflow: '',
  }
}

export function toConnectorForm(connector: IssueConnector): ConnectorForm {
  return {
    type: connector.type,
    name: connector.name,
    status: connector.status,
    base_url: connector.config.base_url,
    auth_token: connector.config.auth_token,
    project_ref: connector.config.project_ref,
    poll_interval: connector.config.poll_interval,
    sync_direction: connector.config.sync_direction,
    labels: connector.config.filters.labels.join(', '),
    exclude_labels: connector.config.filters.exclude_labels.join(', '),
    states: connector.config.filters.states.join(', '),
    authors: connector.config.filters.authors.join(', '),
    status_mapping: Object.entries(connector.config.status_mapping)
      .map(([externalStatus, internalStatus]) => `${externalStatus}=${internalStatus}`)
      .join('\n'),
    webhook_secret: connector.config.webhook_secret,
    auto_workflow: connector.config.auto_workflow,
  }
}

export function toConnectorInput(form: ConnectorForm): ConnectorInput {
  return {
    type: form.type,
    name: form.name.trim(),
    status: form.status,
    config: {
      type: form.type,
      base_url: form.base_url.trim(),
      auth_token: form.auth_token.trim(),
      project_ref: form.project_ref.trim(),
      poll_interval: form.poll_interval.trim() || '5m',
      sync_direction: form.sync_direction,
      filters: {
        labels: parseList(form.labels),
        exclude_labels: parseList(form.exclude_labels),
        states: parseList(form.states),
        authors: parseList(form.authors),
      },
      status_mapping: parseStatusMapping(form.status_mapping),
      webhook_secret: form.webhook_secret.trim(),
      auto_workflow: form.auto_workflow.trim(),
    },
  }
}

export function parseConnectorListPayload(raw: unknown): IssueConnector[] {
  const source = asRecord(raw)
  const items = Array.isArray(source.connectors) ? source.connectors : []
  return items.map(parseConnector).filter((item): item is IssueConnector => item !== null)
}

export function parseConnectorPayload(raw: unknown): IssueConnector | null {
  const source = asRecord(raw)
  return parseConnector(source.connector)
}

export function parseConnector(raw: unknown): IssueConnector | null {
  const source = asRecord(raw)
  const id = readString(source, ['id'])
  if (!id) {
    return null
  }

  const type = parseConnectorType(readString(source, ['type'])) ?? 'custom'
  const status = parseConnectorStatus(readString(source, ['status'])) ?? 'active'
  const config = parseConnectorConfig(readValue(source, ['config']), type)

  return {
    id,
    project_id: readString(source, ['project_id', 'projectId', 'ProjectID']),
    type,
    name: readString(source, ['name']),
    status,
    config,
    last_sync_at: readOptionalString(source, ['last_sync_at', 'lastSyncAt', 'LastSyncAt']),
    last_error: readString(source, ['last_error', 'lastError', 'LastError']),
    stats: parseConnectorStats(readValue(source, ['stats'])),
  }
}

function parseConnectorConfig(raw: unknown, fallbackType: ConnectorType): ConnectorConfig {
  const source = asRecord(raw)
  const type = parseConnectorType(readString(source, ['type'])) ?? fallbackType

  return {
    type,
    base_url: readString(source, ['base_url', 'baseURL', 'BaseURL']),
    auth_token: readString(source, ['auth_token', 'authToken', 'AuthToken']),
    project_ref: readString(source, ['project_ref', 'projectRef', 'ProjectRef']),
    poll_interval: readString(source, ['poll_interval', 'pollInterval', 'PollInterval']) || '5m',
    sync_direction:
      parseSyncDirection(
        readString(source, ['sync_direction', 'syncDirection', 'SyncDirection']),
      ) ?? 'bidirectional',
    filters: parseConnectorFilters(readValue(source, ['filters'])),
    status_mapping: parseStatusMappingRecord(
      readValue(source, ['status_mapping', 'statusMapping', 'StatusMapping']),
    ),
    webhook_secret: readString(source, ['webhook_secret', 'webhookSecret', 'WebhookSecret']),
    auto_workflow: readString(source, ['auto_workflow', 'autoWorkflow', 'AutoWorkflow']),
  }
}

function parseConnectorFilters(raw: unknown): ConnectorFilters {
  const source = asRecord(raw)
  return {
    labels: parseStringList(readValue(source, ['labels'])),
    exclude_labels: parseStringList(
      readValue(source, ['exclude_labels', 'excludeLabels', 'ExcludeLabels']),
    ),
    states: parseStringList(readValue(source, ['states'])),
    authors: parseStringList(readValue(source, ['authors'])),
  }
}

function parseConnectorStats(raw: unknown): ConnectorStats {
  const source = asRecord(raw)
  return {
    total_synced: readNumber(source, ['total_synced', 'totalSynced', 'TotalSynced']),
    synced_24h: readNumber(source, ['synced_24h', 'synced24h', 'Synced24h']),
    failed_count: readNumber(source, ['failed_count', 'failedCount', 'FailedCount']),
  }
}

function parseStatusMapping(source: string) {
  const entries = source
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const separator = line.includes('=') ? '=' : ':'
      const [from, ...rest] = line.split(separator)
      return [from?.trim() ?? '', rest.join(separator).trim()] as const
    })
    .filter(([from, to]) => from && to)

  return Object.fromEntries(entries)
}

function parseStatusMappingRecord(raw: unknown): Record<string, string> {
  const source = asRecord(raw)
  const result: Record<string, string> = {}

  for (const [key, value] of Object.entries(source)) {
    const nextKey = key.trim()
    const nextValue = typeof value === 'string' ? value.trim() : ''
    if (!nextKey || !nextValue) {
      continue
    }
    result[nextKey] = nextValue
  }

  return result
}

function parseList(source: string) {
  return source
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseStringList(raw: unknown) {
  if (!Array.isArray(raw)) {
    return []
  }

  return raw.map((item) => (typeof item === 'string' ? item.trim() : '')).filter(Boolean)
}

function parseConnectorType(raw: string): ConnectorType | null {
  return connectorTypes.find((item) => item === raw.trim()) ?? null
}

function parseConnectorStatus(raw: string): ConnectorStatus | null {
  return connectorStatuses.find((item) => item === raw.trim()) ?? null
}

function parseSyncDirection(raw: string): SyncDirection | null {
  return syncDirections.find((item) => item === raw.trim()) ?? null
}

function asRecord(raw: unknown): Record<string, unknown> {
  return raw && typeof raw === 'object' && !Array.isArray(raw)
    ? (raw as Record<string, unknown>)
    : {}
}

function readValue(source: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    if (key in source) {
      return source[key]
    }
  }

  return undefined
}

function readString(source: Record<string, unknown>, keys: string[]) {
  const value = readValue(source, keys)
  return typeof value === 'string' ? value.trim() : ''
}

function readOptionalString(source: Record<string, unknown>, keys: string[]) {
  const value = readValue(source, keys)
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function readNumber(source: Record<string, unknown>, keys: string[]) {
  const value = readValue(source, keys)
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}
