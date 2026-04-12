import type {
  ProviderConfig,
  ProviderDraft,
  ProviderDraftParseResult,
  ProviderAdapterType,
  ProviderPermissionProfile,
} from './types'
import {
  TOKENS_PER_MILLION,
  createCustomFlatPricingConfig,
  formatPricingPerMillion,
  parseProviderPricingConfig,
} from './provider-pricing'
import { parseSecretBindings, stringifySecretBindingsDraft } from './provider-secret-bindings'

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
    reasoningEffort: '',
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
    reasoningEffort: provider.reasoningEffort ?? '',
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
      reasoning_effort: draft.reasoningEffort.trim(),
      model_temperature: modelTemperature.value,
      model_max_tokens: modelMaxTokens.value,
      max_parallel_runs: maxParallelRuns.value,
      cost_per_input_token: costPerInputToken.value,
      cost_per_output_token: costPerOutputToken.value,
      pricing_config: pricingConfig,
    },
  }
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
