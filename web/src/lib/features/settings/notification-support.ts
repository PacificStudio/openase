export type JSONObject = Record<string, unknown>

type ParseSuccess<T> = {
  ok: true
  value: T
}

type ParseFailure = {
  ok: false
  error: string
}

export type ParseResult<T> = ParseSuccess<T> | ParseFailure

export function formatJSONObject(value: unknown): string {
  const objectValue = isJSONObject(value) ? value : {}
  return JSON.stringify(objectValue, null, 2)
}

export function parseJSONObject(rawText: string, fieldLabel: string): ParseResult<JSONObject> {
  const normalizedText = rawText.trim()
  if (normalizedText === '') {
    return { ok: true, value: {} }
  }

  try {
    const parsed = JSON.parse(normalizedText)
    if (!isJSONObject(parsed)) {
      return { ok: false, error: `${fieldLabel} must be a JSON object.` }
    }
    return { ok: true, value: parsed }
  } catch {
    return { ok: false, error: `${fieldLabel} must be valid JSON.` }
  }
}

function isJSONObject(value: unknown): value is JSONObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
