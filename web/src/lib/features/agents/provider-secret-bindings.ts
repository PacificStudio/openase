import type { ProviderAdapterType, ProviderSecretBinding } from './types'

const providerSecretBindingEnvVarHints: Record<ProviderAdapterType, string[]> = {
  'claude-code-cli': ['ANTHROPIC_API_KEY'],
  'codex-app-server': ['OPENAI_API_KEY'],
  'gemini-cli': ['GEMINI_API_KEY', 'GOOGLE_API_KEY'],
  custom: [],
}

export function requiredProviderSecretBindingEnvVars(adapterType: ProviderAdapterType | string) {
  return adapterType in providerSecretBindingEnvVarHints
    ? [...providerSecretBindingEnvVarHints[adapterType as ProviderAdapterType]]
    : []
}

export function stringifySecretBindingsDraft(bindings: ProviderSecretBinding[]): string {
  const configuredBindings = bindings
    .filter((binding) => binding.configured && binding.source === 'binding')
    .reduce<Record<string, string>>((result, binding) => {
      result[binding.envVarKey] = binding.bindingKey
      return result
    }, {})

  return Object.keys(configuredBindings).length > 0
    ? JSON.stringify(configuredBindings, null, 2)
    : ''
}

export function parseSecretBindings(raw: string):
  | {
      ok: true
      value: Array<{ env_var_key: string; binding_key: string }>
    }
  | {
      ok: false
      error: string
    } {
  const text = raw.trim()
  if (!text) {
    return { ok: true, value: [] }
  }

  try {
    const parsed = JSON.parse(text) as unknown
    if (Array.isArray(parsed)) {
      return parseSecretBindingArray(parsed)
    }
    if (parsed && typeof parsed === 'object') {
      return parseSecretBindingMap(parsed as Record<string, unknown>)
    }
    return {
      ok: false,
      error: 'Secret bindings must be a JSON object or an array of binding entries.',
    }
  } catch {
    return { ok: false, error: 'Secret bindings must be valid JSON.' }
  }
}

function parseSecretBindingArray(raw: unknown[]):
  | {
      ok: true
      value: Array<{ env_var_key: string; binding_key: string }>
    }
  | {
      ok: false
      error: string
    } {
  const bindings: Array<{ env_var_key: string; binding_key: string }> = []
  for (const [index, item] of raw.entries()) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      return {
        ok: false,
        error: `Secret bindings[${index}] must be an object with env_var_key and binding_key.`,
      }
    }
    const envVarKey = normalizeSecretBindingName((item as Record<string, unknown>).env_var_key)
    if (!envVarKey.ok) {
      return { ok: false, error: `Secret bindings[${index}].env_var_key ${envVarKey.error}` }
    }
    const bindingKey = normalizeSecretBindingName((item as Record<string, unknown>).binding_key)
    if (!bindingKey.ok) {
      return { ok: false, error: `Secret bindings[${index}].binding_key ${bindingKey.error}` }
    }
    bindings.push({ env_var_key: envVarKey.value, binding_key: bindingKey.value })
  }
  return { ok: true, value: bindings }
}

function parseSecretBindingMap(raw: Record<string, unknown>):
  | {
      ok: true
      value: Array<{ env_var_key: string; binding_key: string }>
    }
  | {
      ok: false
      error: string
    } {
  const bindings: Array<{ env_var_key: string; binding_key: string }> = []
  for (const [rawEnvVarKey, rawBindingKey] of Object.entries(raw)) {
    const envVarKey = normalizeSecretBindingName(rawEnvVarKey)
    if (!envVarKey.ok) {
      return { ok: false, error: `Secret binding key ${envVarKey.error}` }
    }
    const bindingKey = normalizeSecretBindingName(rawBindingKey)
    if (!bindingKey.ok) {
      return { ok: false, error: `Secret binding ${envVarKey.value} ${bindingKey.error}` }
    }
    bindings.push({ env_var_key: envVarKey.value, binding_key: bindingKey.value })
  }
  return { ok: true, value: bindings }
}

function normalizeSecretBindingName(
  raw: unknown,
): { ok: true; value: string } | { ok: false; error: string } {
  if (typeof raw !== 'string') {
    return { ok: false, error: 'must be a string.' }
  }
  const value = raw.trim().toUpperCase()
  if (!value) {
    return { ok: false, error: 'must not be empty.' }
  }
  if (!/^[A-Z][A-Z0-9_]{0,127}$/.test(value)) {
    return { ok: false, error: 'must match ^[A-Z][A-Z0-9_]{0,127}$.' }
  }
  return { ok: true, value }
}
