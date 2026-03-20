const API_BASE = ''

type FetchOptions = {
  params?: Record<string, string>
  body?: unknown
  signal?: AbortSignal
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public detail: string,
  ) {
    super(detail)
  }
}

async function request<T>(method: string, path: string, opts: FetchOptions = {}): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (opts.params) {
    for (const [k, v] of Object.entries(opts.params)) {
      url.searchParams.set(k, v)
    }
  }

  const headers: Record<string, string> = { 'Content-Type': 'application/json' }

  const res = await fetch(url.toString(), {
    method,
    headers,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
    signal: opts.signal,
    credentials: 'same-origin',
  })

  if (!res.ok) {
    const detail = await res.text().catch(() => res.statusText)
    throw new ApiError(res.status, detail)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string, opts?: FetchOptions) => request<T>('GET', path, opts),
  post: <T>(path: string, opts?: FetchOptions) => request<T>('POST', path, opts),
  put: <T>(path: string, opts?: FetchOptions) => request<T>('PUT', path, opts),
  patch: <T>(path: string, opts?: FetchOptions) => request<T>('PATCH', path, opts),
  delete: <T>(path: string, opts?: FetchOptions) => request<T>('DELETE', path, opts),
}
