import {
  parseConnector,
  toConnectorInput,
  type ConnectorForm,
  type ConnectorInput,
  type ConnectorStatus,
  type IssueConnector,
} from './types'

const storageKeyPrefix = 'openase.connectors.'

export function readLocalConnectors(projectId: string) {
  if (typeof window === 'undefined') {
    return [] as IssueConnector[]
  }

  const raw = window.localStorage.getItem(storageKey(projectId))
  if (!raw) {
    return []
  }

  const items = JSON.parse(raw) as unknown
  if (!Array.isArray(items)) {
    return []
  }

  return items.map(parseConnector).filter((item): item is IssueConnector => item !== null)
}

export function writeLocalConnectors(projectId: string, connectors: IssueConnector[]) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(storageKey(projectId), JSON.stringify(connectors))
}

export function upsertLocalConnector(
  projectId: string,
  connectorId: string,
  input: ConnectorInput,
): IssueConnector[] {
  const connectors = readLocalConnectors(projectId)
  const existing = connectors.find((item) => item.id === connectorId) ?? null
  const nextConnector = {
    id: existing?.id ?? crypto.randomUUID(),
    project_id: projectId,
    type: input.type,
    name: input.name,
    status: input.status,
    config: input.config,
    last_sync_at: existing?.last_sync_at ?? null,
    last_error: existing?.last_error ?? '',
    stats: existing?.stats ?? {
      total_synced: 0,
      synced_24h: 0,
      failed_count: 0,
    },
  } satisfies IssueConnector

  const nextItems = existing
    ? connectors.map((item) => (item.id === connectorId ? nextConnector : item))
    : [nextConnector, ...connectors]
  writeLocalConnectors(projectId, nextItems)
  return nextItems
}

export function deleteLocalConnector(projectId: string, connectorId: string) {
  const nextItems = readLocalConnectors(projectId).filter((item) => item.id !== connectorId)
  writeLocalConnectors(projectId, nextItems)
  return nextItems
}

export function setLocalConnectorStatus(
  projectId: string,
  connectorId: string,
  status: ConnectorStatus,
) {
  const nextItems = readLocalConnectors(projectId).map((item) =>
    item.id === connectorId
      ? {
          ...item,
          status,
          last_error: status === 'error' ? item.last_error : '',
        }
      : item,
  )
  writeLocalConnectors(projectId, nextItems)
  return nextItems
}

export function simulateLocalSync(projectId: string, connectorId: string) {
  const now = new Date().toISOString()
  const nextItems = readLocalConnectors(projectId).map((item) => {
    if (item.id !== connectorId) {
      return item
    }

    return {
      ...item,
      status: 'active' as const,
      last_sync_at: now,
      last_error: '',
      stats: {
        total_synced: item.stats.total_synced + 1,
        synced_24h: item.stats.synced_24h + 1,
        failed_count: item.stats.failed_count,
      },
    }
  })
  writeLocalConnectors(projectId, nextItems)
  return nextItems
}

export function simulateLocalTest(projectId: string, connectorId: string) {
  const nextItems = readLocalConnectors(projectId).map((item) => {
    if (item.id !== connectorId) {
      return item
    }

    return {
      ...item,
      status: 'active' as const,
      last_error: '',
    }
  })
  writeLocalConnectors(projectId, nextItems)
  return nextItems
}

export function duplicateFormAsLocalConnector(projectId: string, form: ConnectorForm) {
  return upsertLocalConnector(projectId, '', toConnectorInput(form))
}

function storageKey(projectId: string) {
  return `${storageKeyPrefix}${projectId}`
}
