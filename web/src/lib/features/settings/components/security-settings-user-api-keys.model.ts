export function toRFC3339Local(localValue: string): string | undefined {
  const trimmed = localValue.trim()
  if (!trimmed) return undefined
  const value = new Date(trimmed)
  return Number.isNaN(value.getTime()) ? undefined : value.toISOString()
}

export function formatUserAPIKeyTime(value?: string | null): string {
  if (!value) return 'Never'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
