export function parseArray<T>(raw: unknown, parseItem: (item: unknown) => T | null): T[] {
  if (!Array.isArray(raw)) {
    return []
  }

  return raw.map(parseItem).filter((item): item is T => item !== null)
}

export function parseStringArray(raw: unknown): string[] {
  if (!Array.isArray(raw)) {
    return []
  }

  return raw.map((item) => (typeof item === 'string' ? item : '')).filter(Boolean)
}

export function parseUnknownRecord(raw: unknown): Record<string, unknown> {
  if (!isRecord(raw)) {
    return {}
  }

  return { ...raw }
}

export function asRecord(raw: unknown): Record<string, unknown> {
  if (!isRecord(raw)) {
    return {}
  }

  return raw
}

export function readString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'string' ? value : ''
}

export function readNullableString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  if (value === null) {
    return null
  }

  return typeof value === 'string' ? value : null
}

export function readNumber(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'number' ? value : 0
}

export function readBoolean(source: Record<string, unknown>, key: string) {
  return source[key] === true
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
