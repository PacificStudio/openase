import { parseConnectorListPayload, parseConnectorPayload, type ConnectorInput } from './types'

export class ConnectorAPIError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ConnectorAPIError'
    this.status = status
  }
}

export async function listProjectConnectors(projectId: string) {
  return request(`/api/v1/projects/${projectId}/connectors`, {}, parseConnectorListPayload)
}

export async function createConnector(projectId: string, input: ConnectorInput) {
  const connector = await request(
    `/api/v1/projects/${projectId}/connectors`,
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
    parseConnectorPayload,
  )
  if (!connector) {
    throw new ConnectorAPIError(502, 'connector create response did not include a connector')
  }

  return connector
}

export async function updateConnector(connectorId: string, input: ConnectorInput) {
  const connector = await request(
    `/api/v1/connectors/${connectorId}`,
    {
      method: 'PATCH',
      body: JSON.stringify(input),
    },
    parseConnectorPayload,
  )
  if (!connector) {
    throw new ConnectorAPIError(502, 'connector update response did not include a connector')
  }

  return connector
}

export async function deleteConnector(connectorId: string) {
  await request(
    `/api/v1/connectors/${connectorId}`,
    {
      method: 'DELETE',
    },
    () => undefined,
  )
}

export async function syncConnector(connectorId: string) {
  await request(
    `/api/v1/connectors/${connectorId}/sync`,
    {
      method: 'POST',
    },
    () => undefined,
  )
}

export async function testConnector(connectorId: string) {
  await request(
    `/api/v1/connectors/${connectorId}/test`,
    {
      method: 'POST',
    },
    () => undefined,
  )
}

export function isConnectorAPIUnavailable(error: unknown) {
  return error instanceof ConnectorAPIError && [404, 405, 501].includes(error.status)
}

async function request<T>(
  path: string,
  init: RequestInit,
  parse: (payload: unknown) => T,
): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('content-type')) {
    headers.set('content-type', 'application/json')
  }

  const response = await fetch(path, {
    ...init,
    headers,
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    const source =
      payload && typeof payload === 'object' && !Array.isArray(payload)
        ? (payload as Record<string, unknown>)
        : {}
    const message =
      (typeof source.message === 'string' && source.message) ||
      (typeof source.error === 'string' && source.error) ||
      `request failed with status ${response.status}`
    throw new ConnectorAPIError(response.status, message)
  }

  return parse(payload)
}
