import { type IssueConnectorRecord, type getIssueConnectorStats } from '$lib/api/openase'

export type ConnectorDraft = {
  name: string
  projectRef: string
  status: string
  syncDirection: string
  pollInterval: string
  baseURL: string
  labelFilter: string
  statusMapping: string
  authToken: string
  webhookSecret: string
  autoWorkflow: string
}

export type ConnectorStats = Awaited<ReturnType<typeof getIssueConnectorStats>>['stats']

export type ConnectorsSettingsUI = {
  connectors: IssueConnectorRecord[]
  statsByConnectorId: Record<string, ConnectorStats>
  loading: boolean
  saving: boolean
  busyConnectorId: string
  editorMode: 'create' | 'edit'
  editingConnectorId: string
  draft: ConnectorDraft
}

export function createEmptyConnectorDraft(): ConnectorDraft {
  return {
    name: '',
    projectRef: '',
    status: 'active',
    syncDirection: 'bidirectional',
    pollInterval: '5m',
    baseURL: '',
    labelFilter: 'openase',
    statusMapping: 'open=Todo\nclosed=Done',
    authToken: '',
    webhookSecret: '',
    autoWorkflow: '',
  }
}

export function connectorToDraft(connector: IssueConnectorRecord): ConnectorDraft {
  return {
    name: connector.name,
    projectRef: connector.config.project_ref,
    status: connector.status,
    syncDirection: connector.config.sync_direction,
    pollInterval: connector.config.poll_interval,
    baseURL: connector.config.base_url,
    labelFilter: connector.config.filters.labels.join(', '),
    statusMapping: Object.entries(connector.config.status_mapping)
      .map(([externalStatus, localStatus]) => `${externalStatus}=${localStatus}`)
      .join('\n'),
    authToken: '',
    webhookSecret: '',
    autoWorkflow: connector.config.auto_workflow,
  }
}

export function parseConnectorCSV(raw: string) {
  return raw
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean)
}

export function parseConnectorStatusMapping(raw: string) {
  const mapping: Record<string, string> = {}
  const lines = raw
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)

  for (const line of lines) {
    const separator = line.indexOf('=')
    if (separator <= 0 || separator === line.length - 1) {
      return { ok: false as const, error: `Invalid status mapping line: ${line}` }
    }

    const externalStatus = line.slice(0, separator).trim()
    const localStatus = line.slice(separator + 1).trim()
    if (!externalStatus || !localStatus) {
      return { ok: false as const, error: `Invalid status mapping line: ${line}` }
    }
    mapping[externalStatus] = localStatus
  }

  return { ok: true as const, value: mapping }
}

export function createConnectorsSettingsUI(): ConnectorsSettingsUI {
  return {
    connectors: [],
    statsByConnectorId: {},
    loading: false,
    saving: false,
    busyConnectorId: '',
    editorMode: 'create',
    editingConnectorId: '',
    draft: createEmptyConnectorDraft(),
  }
}

export function connectorStatsFallback(connector: IssueConnectorRecord): ConnectorStats {
  return {
    connector_id: connector.id,
    status: connector.status,
    last_sync_at: connector.last_sync_at ?? null,
    last_error: connector.last_error,
    stats: connector.stats,
  }
}
