import { authStore } from '$lib/stores/auth.svelte'

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
    public code?: string,
    public details?: unknown,
  ) {
    super(detail)
  }
}

function isMutatingMethod(method: string) {
  return method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS'
}

export function buildRequestHeaders(method: string, initialHeaders: Record<string, string> = {}) {
  const headers = { ...initialHeaders }
  if (isMutatingMethod(method)) {
    const csrfToken = authStore.csrfToken.trim()
    if (csrfToken) {
      headers['X-OpenASE-CSRF'] = csrfToken
    }
  }
  return headers
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

  let headers: Record<string, string> = {}
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  headers = buildRequestHeaders(method, headers)

  const res = await fetch(url.toString(), {
    method,
    headers,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
    signal: opts.signal,
    credentials: 'same-origin',
  })

  if (!res.ok) {
    const payload = await readErrorPayload(res)
    throw new ApiError(res.status, payload.detail, payload.code, payload.details)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

async function readErrorPayload(
  res: Response,
): Promise<{ detail: string; code?: string; details?: unknown }> {
  try {
    const contentType = res.headers.get('content-type') ?? ''
    if (contentType.includes('application/json')) {
      const payload = (await res.json()) as {
        message?: string
        detail?: string
        error?: string
        code?: string
        details?: unknown
      }
      return {
        detail:
          payload.message || payload.detail || payload.error || payload.code || res.statusText,
        code: payload.code,
        details: payload.details,
      }
    }
  } catch {
    return { detail: res.statusText }
  }

  return {
    detail: await res.text().catch(() => res.statusText),
  }
}

export const api = {
  get: <T>(path: string, opts?: FetchOptions) => request<T>('GET', path, opts),
  post: <T>(path: string, opts?: FetchOptions) => request<T>('POST', path, opts),
  put: <T>(path: string, opts?: FetchOptions) => request<T>('PUT', path, opts),
  patch: <T>(path: string, opts?: FetchOptions) => request<T>('PATCH', path, opts),
  delete: <T>(path: string, opts?: FetchOptions) => request<T>('DELETE', path, opts),
}
