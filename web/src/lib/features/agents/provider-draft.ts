import type {
  ProviderConfig,
  ProviderDraft,
  ProviderDraftParseResult,
  ProviderAdapterType,
  ProviderPermissionProfile,
  ProviderSecretBinding,
} from './types'
import {
  TOKENS_PER_MILLION,
  createCustomFlatPricingConfig,
  formatPricingPerMillion,
  parseProviderPricingConfig,
} from './provider-pricing'

export const providerAdapterOptions: Array<{ value: ProviderAdapterType; label: string }> = [
  { value: 'claude-code-cli', label: 'Claude Code CLI' },
  { value: 'codex-app-server', label: 'Codex App Server' },
  { value: 'gemini-cli', label: 'Gemini CLI' },
  { value: 'custom', label: 'Custom' },
]

export const providerPermissionProfileOptions: Array<{
  value: ProviderPermissionProfile
  label: string
  description: string
}> = [
  {
    value: 'unrestricted',
    label: 'Unrestricted',
    description: 'Auto-approve all actions. Codex also disables sandbox boundaries.',
  },
  {
    value: 'standard',
    label: 'Standard',
    description: 'Do not inject provider-managed bypass flags. Use only if prompts are acceptable.',
  },
]

const providerSecretBindingEnvVarHints: Record<ProviderAdapterType, string[]> = {
  'claude-code-cli': ['ANTHROPIC_API_KEY'],
  'codex-app-server': ['OPENAI_API_KEY'],
  'gemini-cli': ['GEMINI_API_KEY', 'GOOGLE_API_KEY'],
  custom: [],
}

export function createEmptyProviderDraft(): ProviderDraft {
  return {
    machineId: '',
    name: '',
    adapterType: 'custom',
    permissionProfile: 'unrestricted',
    cliCommand: '',
    cliArgs: '',
    authConfig: '',
    secretBindings: '',
    modelName: '',
    modelTemperature: '0',
    modelMaxTokens: '1',
    maxParallelRuns: '',
    costPerInputToken: '0',
    costPerOutputToken: '0',
    pricingConfig: '',
  }
}

export function providerToDraft(provider: ProviderConfig): ProviderDraft {
  return {
    machineId: provider.machineId,
    name: provider.name,
    adapterType: provider.adapterType,
    permissionProfile: provider.permissionProfile,
    cliCommand: provider.cliCommand,
    cliArgs: provider.cliArgs.join('\n'),
    authConfig:
      Object.keys(provider.authConfig).length > 0
        ? JSON.stringify(provider.authConfig, null, 2)
        : '',
    secretBindings: stringifySecretBindingsDraft(provider.secretBindings),
    modelName: provider.modelName,
    modelTemperature: String(provider.modelTemperature),
    modelMaxTokens: String(provider.modelMaxTokens),
    maxParallelRuns: provider.maxParallelRuns > 0 ? String(provider.maxParallelRuns) : '',
    costPerInputToken: formatPricingPerMillion(provider.costPerInputToken),
    costPerOutputToken: formatPricingPerMillion(provider.costPerOutputToken),
    pricingConfig: provider.pricingConfig ? JSON.stringify(provider.pricingConfig) : '',
  }
}

export function parseProviderDraft(draft: ProviderDraft): ProviderDraftParseResult {
  const machineId = draft.machineId.trim()
  if (!machineId) {
    return { ok: false, error: 'Execution machine is required.' }
  }

  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Provider name is required.' }
  }

  const adapterType = parseProviderAdapterType(draft.adapterType)
  if (!adapterType) {
    return {
      ok: false,
      error: `Adapter type must be one of ${providerAdapterOptions.map((option) => option.value).join(', ')}.`,
    }
  }

  const permissionProfile = parseProviderPermissionProfile(draft.permissionProfile)
  if (!permissionProfile) {
    return {
      ok: false,
      error: `Permission mode must be one of ${providerPermissionProfileOptions.map((option) => option.value).join(', ')}.`,
    }
  }

  const modelName = draft.modelName.trim()
  if (!modelName) {
    return { ok: false, error: 'Model name is required.' }
  }

  const modelTemperature = parseNonNegativeNumber('Model temperature', draft.modelTemperature)
  if (!modelTemperature.ok) {
    return modelTemperature
  }

  const modelMaxTokens = parsePositiveInteger('Model max tokens', draft.modelMaxTokens)
  if (!modelMaxTokens.ok) {
    return modelMaxTokens
  }

  const maxParallelRuns = parseOptionalPositiveInteger('Max parallel runs', draft.maxParallelRuns)
  if (!maxParallelRuns.ok) {
    return maxParallelRuns
  }

  const costPerInputToken = parsePricingPerMillion(
    'Input pricing (USD per 1M tokens)',
    draft.costPerInputToken,
  )
  if (!costPerInputToken.ok) {
    return costPerInputToken
  }

  const costPerOutputToken = parsePricingPerMillion(
    'Output pricing (USD per 1M tokens)',
    draft.costPerOutputToken,
  )
  if (!costPerOutputToken.ok) {
    return costPerOutputToken
  }

  const authConfig = parseAuthConfig(draft.authConfig)
  if (!authConfig.ok) {
    return authConfig
  }

  const secretBindings = parseSecretBindings(draft.secretBindings)
  if (!secretBindings.ok) {
    return secretBindings
  }

  const pricingConfig =
    parseProviderPricingConfig(draft.pricingConfig) ??
    createCustomFlatPricingConfig(costPerInputToken.value, costPerOutputToken.value)

  return {
    ok: true,
    value: {
      machine_id: machineId,
      name,
      adapter_type: adapterType,
      permission_profile: permissionProfile,
      cli_command: draft.cliCommand.trim(),
      cli_args: splitLines(draft.cliArgs),
      auth_config: authConfig.value,
      secret_bindings: secretBindings.value,
      model_name: modelName,
      model_temperature: modelTemperature.value,
      model_max_tokens: modelMaxTokens.value,
      max_parallel_runs: maxParallelRuns.value,
      cost_per_input_token: costPerInputToken.value,
      cost_per_output_token: costPerOutputToken.value,
      pricing_config: pricingConfig,
    },
  }
}

export function requiredProviderSecretBindingEnvVars(adapterType: string): string[] {
  const parsed = parseProviderAdapterType(adapterType)
  return parsed ? [...providerSecretBindingEnvVarHints[parsed]] : []
}

function parseProviderAdapterType(raw: string): ProviderAdapterType | null {
  const value = raw.trim().toLowerCase()
  return providerAdapterOptions.some((option) => option.value === value)
    ? (value as ProviderAdapterType)
    : null
}

function parseProviderPermissionProfile(raw: string): ProviderPermissionProfile | null {
  const value = raw.trim().toLowerCase()
  return providerPermissionProfileOptions.some((option) => option.value === value)
    ? (value as ProviderPermissionProfile)
    : null
}

function splitLines(raw: string): string[] {
  return raw
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)
}

function parseNonNegativeNumber(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  const value = Number(raw.trim())
  if (!Number.isFinite(value) || value < 0) {
    return { ok: false, error: `${label} must be a number greater than or equal to 0.` }
  }
  return { ok: true, value }
}

function parsePricingPerMillion(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  if (!raw.trim()) {
    return { ok: true, value: 0 }
  }

  const parsed = parseNonNegativeNumber(label, raw)
  if (!parsed.ok) {
    return parsed
  }

  return {
    ok: true,
    value: parsed.value / TOKENS_PER_MILLION,
  }
}

function parsePositiveInteger(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  const value = Number(raw.trim())
  if (!Number.isInteger(value) || value <= 0) {
    return { ok: false, error: `${label} must be a positive integer.` }
  }
  return { ok: true, value }
}

function parseOptionalPositiveInteger(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  const value = raw.trim()
  if (!value) {
    return { ok: true, value: 0 }
  }

  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return { ok: false, error: `${label} must be a positive integer.` }
  }

  return { ok: true, value: parsed }
}

function parseAuthConfig(
  raw: string,
): { ok: true; value: Record<string, unknown> } | { ok: false; error: string } {
  const text = raw.trim()
  if (!text) {
    return { ok: true, value: {} }
  }

  try {
    const parsed = JSON.parse(text) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { ok: false, error: 'Auth config must be a JSON object.' }
    }
    return { ok: true, value: parsed as Record<string, unknown> }
  } catch {
    return { ok: false, error: 'Auth config must be valid JSON.' }
  }
}

function stringifySecretBindingsDraft(bindings: ProviderSecretBinding[]): string {
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

function parseSecretBindings(raw: string):
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
