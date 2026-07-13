import { i18nStore } from '$lib/i18n/store.svelte'
import { formatMachineRelativeTime } from '../machine-i18n'
import type {
  RuntimeStatusLike,
  TruthyState,
  MachineLevelStateInput,
} from './machine-health-panel-types'

export function toTruthyState(value: boolean | undefined): TruthyState {
  if (value === undefined) return 'unknown'
  return value ? 'yes' : 'no'
}

export function checkedAtLabel(value: string | undefined): string {
  return value
    ? i18nStore.t('machines.machineHealthPanel.dynamic.checkedAt', {
        time: formatMachineRelativeTime(value),
      })
    : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet')
}

export function runtimeLabel(runtime: RuntimeStatusLike): string {
  switch (runtime.name) {
    case 'claude_code':
      return 'Claude Code'
    case 'codex':
      return 'Codex'
    case 'gemini':
      return 'Gemini'
    default:
      return runtime.name
  }
}

export function levelState(level: MachineLevelStateInput | undefined): string {
  switch (level?.state) {
    case 'healthy':
      return 'ok'
    case 'degraded':
      return 'warn'
    case 'failed':
      return 'error'
  }
  if (!level) return 'unknown'
  if (level.error) return 'error'
  if (level.checkedAt) return 'ok'
  return 'unknown'
}

export function stateBadgeVariant(state: string): 'secondary' | 'destructive' | 'outline' {
  switch (state) {
    case 'ok':
      return 'secondary'
    case 'warn':
      return 'outline'
    case 'error':
      return 'destructive'
    default:
      return 'outline'
  }
}

export function stateLabel(state: string): string {
  switch (state) {
    case 'ok':
      return i18nStore.t('machines.machineHealthPanel.status.healthy')
    case 'warn':
      return i18nStore.t('machines.machineHealthPanel.status.degraded')
    case 'error':
      return i18nStore.t('machines.machineHealthPanel.status.failed')
    default:
      return i18nStore.t('machines.machineHealthPanel.status.unknown')
  }
}
