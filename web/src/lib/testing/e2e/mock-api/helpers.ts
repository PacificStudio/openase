import type { JsonValue } from './constants'

export function clone<T>(value: T): T {
  return structuredClone(value)
}

export function findById(
  items: Record<string, unknown>[],
  id: string,
): Record<string, unknown> | undefined {
  return items.find((item) => item.id === id)
}

export function asString(value: JsonValue | undefined): string | undefined {
  return typeof value === 'string' ? value : undefined
}

export function asNumber(value: JsonValue | undefined): number | undefined {
  return typeof value === 'number' ? value : undefined
}

export function asBoolean(value: JsonValue | undefined): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

export function asObject(value: JsonValue | undefined): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

export function asStringArray(value: JsonValue | undefined): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

export function asObjectArray(value: JsonValue | undefined): Record<string, unknown>[] | null {
  return Array.isArray(value)
    ? value.filter((item): item is Record<string, unknown> => Boolean(asObject(item)))
    : null
}

export function asSecretBindings(
  value: JsonValue | undefined,
): Array<{ env_var_key: string; binding_key: string; configured: boolean; source: string }> {
  return (
    asObjectArray(value)?.map((item) => ({
      env_var_key: asString(item.env_var_key) ?? '',
      binding_key: asString(item.binding_key) ?? '',
      configured: asBoolean(item.configured) ?? false,
      source: asString(item.source) ?? '',
    })) ?? []
  )
}

export function decodeBase64UTF8(value: string): string {
  if (typeof Buffer !== 'undefined') {
    return Buffer.from(value, 'base64').toString('utf8')
  }
  const binary = atob(value)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

export async function readBody<T>(request: Request): Promise<T> {
  const raw = await request.text()
  if (!raw) {
    return {} as T
  }
  return JSON.parse(raw) as T
}

export function jsonResponse(body: JsonValue | Record<string, unknown>, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': 'application/json',
    },
  })
}

export function noContentResponse() {
  return new Response(null, {
    status: 204,
    headers: {
      'content-length': '0',
    },
  })
}

export function notFound(detail: string) {
  return jsonResponse({ detail }, 404)
}
