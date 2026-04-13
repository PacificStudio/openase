import { i18nStore } from '$lib/i18n/store.svelte'
import type { Machine } from '$lib/api/contracts'
import type { MachineDraft, MachineStatus } from './types'
import { normalizeReachabilityMode } from './machine-guidance'

export function normalizeMachineStatus(status: string): MachineStatus {
  if (
    status === 'online' ||
    status === 'offline' ||
    status === 'degraded' ||
    status === 'maintenance'
  ) {
    return status
  }
  return 'maintenance'
}

export function machineStatusLabel(status: string): string {
  switch (normalizeMachineStatus(status)) {
    case 'online':
      return i18nStore.t('machines.status.label.online')
    case 'degraded':
      return i18nStore.t('machines.status.label.degraded')
    case 'offline':
      return i18nStore.t('machines.status.label.offline')
    case 'maintenance':
    default:
      return i18nStore.t('machines.status.label.maintenance')
  }
}

export function machineStatusDescription(status: string): string {
  switch (normalizeMachineStatus(status)) {
    case 'online':
      return i18nStore.t('machines.status.description.online')
    case 'degraded':
      return i18nStore.t('machines.status.description.degraded')
    case 'offline':
      return i18nStore.t('machines.status.description.offline')
    case 'maintenance':
    default:
      return i18nStore.t('machines.status.description.maintenance')
  }
}

export function machineStatusBadgeClass(status: string): string {
  switch (normalizeMachineStatus(status)) {
    case 'online':
      return 'border-emerald-500/30 bg-emerald-500/12 text-emerald-700'
    case 'degraded':
      return 'border-amber-500/30 bg-amber-500/14 text-amber-700'
    case 'offline':
      return 'border-rose-500/30 bg-rose-500/12 text-rose-700'
    case 'maintenance':
    default:
      return 'border-slate-500/20 bg-slate-500/10 text-slate-700'
  }
}

export function filterMachines(machines: Machine[], searchQuery: string): Machine[] {
  const query = searchQuery.trim().toLowerCase()
  if (!query) {
    return machines
  }

  return machines.filter((machine) =>
    [
      machine.name,
      machine.host,
      machine.status,
      machine.reachability_mode,
      machine.execution_mode,
      machine.advertised_endpoint,
      machine.detected_os,
      machine.detected_arch,
      (machine.labels ?? []).join(' '),
      machine.description,
    ]
      .join(' ')
      .toLowerCase()
      .includes(query),
  )
}

export function isLocalMachine(machine: Machine | null | undefined, draft?: MachineDraft): boolean {
  return (
    normalizeReachabilityMode(
      draft?.reachabilityMode ?? machine?.reachability_mode,
      draft?.host ?? machine?.host,
    ) === 'local'
  )
}
