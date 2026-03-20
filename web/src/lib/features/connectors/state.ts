import {
  defaultConnectorForm,
  toConnectorForm,
  type ConnectorForm,
  type IssueConnector,
} from './types'

export function deriveSelectionState(connectors: IssueConnector[], preferredConnectorId: string) {
  const nextSelection =
    connectors.find((item) => item.id === preferredConnectorId) ?? connectors[0] ?? null
  if (!nextSelection) {
    return {
      selectedConnectorId: '',
      form: defaultConnectorForm(),
    }
  }

  return {
    selectedConnectorId: nextSelection.id,
    form: toConnectorForm(nextSelection),
  }
}

export function findConnector(connectors: IssueConnector[], connectorId: string) {
  return connectors.find((item) => item.id === connectorId) ?? null
}

export function deriveUpdatedForm<K extends keyof ConnectorForm>(
  form: ConnectorForm,
  key: K,
  value: ConnectorForm[K],
): ConnectorForm {
  const nextForm = {
    ...form,
    [key]: value,
  }

  if (key === 'type' && value === 'inbound-webhook') {
    return {
      ...nextForm,
      base_url: '',
      auth_token: '',
      project_ref: nextForm.project_ref,
      sync_direction: 'push_only',
    }
  }

  if (key === 'type' && value === 'github' && !nextForm.base_url) {
    return {
      ...nextForm,
      base_url: 'https://api.github.com',
    }
  }

  return nextForm
}
