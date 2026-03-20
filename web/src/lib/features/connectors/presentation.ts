import type { ConnectorStatus, IssueConnector } from './types'

export function statusBadgeVariant(status: string): 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'active':
      return 'secondary'
    case 'error':
      return 'destructive'
    default:
      return 'outline'
  }
}

export function syncDirectionLabel(direction: string) {
  switch (direction) {
    case 'pull_only':
      return 'Pull only'
    case 'push_only':
      return 'Push only'
    default:
      return 'Bidirectional'
  }
}

export function formatTimestamp(value?: string | null) {
  if (!value) {
    return 'Never'
  }

  const parsed = new Date(value)
  return Number.isNaN(parsed.valueOf()) ? value : parsed.toLocaleString()
}

export function connectorSummary(connector: IssueConnector) {
  if (connector.type === 'inbound-webhook') {
    return 'Push ingress only'
  }

  return syncDirectionLabel(connector.config.sync_direction)
}

export function countConnectors(connectors: IssueConnector[], status: ConnectorStatus) {
  return connectors.filter((item) => item.status === status).length
}
