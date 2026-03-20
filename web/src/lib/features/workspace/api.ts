export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('content-type')) {
    headers.set('content-type', 'application/json')
  }

  const response = await fetch(path, {
    ...init,
    headers,
  })

  const payload = (await response.json().catch(() => ({}))) as {
    error?: string
    message?: string
  }
  if (!response.ok) {
    throw new Error(
      payload.message ?? payload.error ?? `request failed with status ${response.status}`,
    )
  }

  return payload as T
}

export function toErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }

  return 'Request failed'
}
