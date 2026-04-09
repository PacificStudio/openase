import { ApiError } from '$lib/api/client'
import { normalizeReturnTo } from '$lib/api/auth'
import { redeemLocalBootstrapAuthorization } from '$lib/api/auth'

export type LocalBootstrapRedeemInput = {
  requestID: string
  code: string
  nonce: string
}

export type LocalBootstrapAuthorizationBundle = LocalBootstrapRedeemInput & {
  returnTo: string
}

export async function redeemLocalBootstrapBrowserSession(
  input: LocalBootstrapRedeemInput,
): Promise<void> {
  await redeemLocalBootstrapAuthorization(input)
}

export function buildLocalBootstrapRedeemPath(input: LocalBootstrapAuthorizationBundle): string {
  const params = new URLSearchParams()
  params.set('request_id', input.requestID)
  params.set('code', input.code)
  params.set('nonce', input.nonce)
  params.set('return_to', normalizeReturnTo(input.returnTo))
  return `/local-bootstrap?${params.toString()}`
}

export function parseLocalBootstrapAuthorizationBundle(
  raw: string,
  fallbackReturnTo: string,
): LocalBootstrapAuthorizationBundle | null {
  const trimmed = normalizeRawAuthorizationInput(raw)
  if (!trimmed) {
    return null
  }

  for (const candidate of authorizationCandidates(trimmed)) {
    const parsed = parseLocalBootstrapCandidate(candidate)
    if (parsed) {
      return {
        requestID: parsed.requestID,
        code: parsed.code,
        nonce: parsed.nonce,
        returnTo: normalizeReturnTo(fallbackReturnTo),
      }
    }
  }

  return null
}

function normalizeRawAuthorizationInput(raw: string): string {
  return raw.trim().replaceAll('\\\\u0026', '&').replaceAll('\\u0026', '&')
}

function authorizationCandidates(raw: string): string[] {
  const candidates = [raw]
  const parsedJSON = parseJSON(raw)
  if (typeof parsedJSON === 'string') {
    candidates.push(parsedJSON)
  } else if (isRecord(parsedJSON)) {
    candidates.push(JSON.stringify(parsedJSON))
    if (typeof parsedJSON.url === 'string') {
      candidates.push(parsedJSON.url)
    }
  }

  const textURLMatch = raw.match(/https?:\/\/\S+/)
  if (textURLMatch?.[0]) {
    candidates.push(textURLMatch[0])
  }

  return candidates
}

function parseLocalBootstrapCandidate(
  candidate: string,
): (LocalBootstrapRedeemInput & { returnTo?: string }) | null {
  const parsedJSON = parseJSON(candidate)
  if (isRecord(parsedJSON)) {
    const parsedFields = parseLocalBootstrapFields(parsedJSON)
    if (parsedFields) {
      return parsedFields
    }
  }

  const querySource = candidate.includes('?')
    ? candidate.slice(candidate.indexOf('?') + 1)
    : candidate
  const parsedQuery = parseLocalBootstrapSearchParams(new URLSearchParams(querySource))
  if (parsedQuery) {
    return parsedQuery
  }

  try {
    const parsedURL = new URL(candidate, 'http://local-bootstrap.invalid')
    return parseLocalBootstrapSearchParams(parsedURL.searchParams)
  } catch {
    return null
  }
}

function parseLocalBootstrapSearchParams(
  params: URLSearchParams,
): (LocalBootstrapRedeemInput & { returnTo?: string }) | null {
  const requestID = params.get('request_id')?.trim() ?? ''
  const code = params.get('code')?.trim() ?? ''
  const nonce = params.get('nonce')?.trim() ?? ''
  if (!requestID || !code || !nonce) {
    return null
  }
  return {
    requestID,
    code,
    nonce,
    returnTo: params.get('return_to')?.trim() ?? '',
  }
}

function parseLocalBootstrapFields(
  raw: Record<string, unknown>,
): (LocalBootstrapRedeemInput & { returnTo?: string }) | null {
  const requestID = readString(raw, 'request_id')
  const code = readString(raw, 'code')
  const nonce = readString(raw, 'nonce')
  if (requestID && code && nonce) {
    return {
      requestID,
      code,
      nonce,
      returnTo: readString(raw, 'return_to'),
    }
  }

  const url = readString(raw, 'url')
  if (!url) {
    return null
  }
  return parseLocalBootstrapCandidate(url)
}

function parseJSON(raw: string): unknown | null {
  try {
    return JSON.parse(raw) as unknown
  } catch {
    return null
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function readString(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  return typeof value === 'string' ? value.trim() : ''
}

export function describeLocalBootstrapRedeemError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.detail || 'Local bootstrap authorization failed.'
  }
  return 'Local bootstrap authorization failed.'
}
