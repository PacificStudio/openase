const API_BASE = ''

type QueryParamValue = string | number | boolean
type QueryParams = Record<string, QueryParamValue | null | undefined>

type FetchOptions = {
  params?: QueryParams
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
      if (v == null) {
        continue
      }
      url.searchParams.set(k, String(v))
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
    const detail = await readErrorDetail(res)
    throw new ApiError(res.status, detail)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

async function readErrorDetail(res: Response) {
  try {
    const contentType = res.headers.get('content-type') ?? ''
    if (contentType.includes('application/json')) {
      const payload = (await res.json()) as {
        message?: string
        detail?: string
        error?: string
        code?: string
      }
      return payload.message || payload.detail || payload.error || payload.code || res.statusText
    }
  } catch {
    return res.statusText
  }

  return res.text().catch(() => res.statusText)
}

export const api = {
  get: <T>(path: string, opts?: FetchOptions) => request<T>('GET', path, opts),
  post: <T>(path: string, opts?: FetchOptions) => request<T>('POST', path, opts),
  put: <T>(path: string, opts?: FetchOptions) => request<T>('PUT', path, opts),
  patch: <T>(path: string, opts?: FetchOptions) => request<T>('PATCH', path, opts),
  delete: <T>(path: string, opts?: FetchOptions) => request<T>('DELETE', path, opts),
}
