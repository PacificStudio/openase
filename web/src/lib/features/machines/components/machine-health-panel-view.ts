import { i18nStore } from '$lib/i18n/store.svelte'
import { friendlyTransportLabel } from '../machine-setup'
import { formatMachineRelativeTime } from '../machine-i18n'
import type { MachineCLIStatus, MachineSnapshot } from '../types'

export type HealthStatCard = {
  label: string
  value: string
  meta: string
}

export type HealthLevelCard = {
  id: string
  label: string
  state: string
  value: string
  meta: string
}

export type TruthyState = 'yes' | 'no' | 'unknown'

export type AuditNetworkEndpoint = {
  name: string
  reachable: TruthyState
}

export type HealthAuditRow =
  | {
      kind: 'git'
      label: string
      installed: TruthyState
      identity: string | null
    }
  | {
      kind: 'gh-cli'
      label: string
      installed: TruthyState
      authStatus: string | null
    }
  | {
      kind: 'network'
      label: string
      endpoints: AuditNetworkEndpoint[]
      auditTimestamp: string | null
    }

export function buildStatCards(snapshot: MachineSnapshot): HealthStatCard[] {
  return [
    {
      label: i18nStore.t('machines.shared.reachability'),
      value:
        snapshot.monitor.l1?.reachable === undefined
          ? i18nStore.t('machines.machineHealthPanel.status.unknown')
          : snapshot.monitor.l1.reachable
            ? i18nStore.t('machines.machineHealthPanel.dynamic.reachable')
            : i18nStore.t('machines.machineHealthPanel.dynamic.unavailable'),
      meta: snapshot.monitor.l1?.latencyMs
        ? `${snapshot.monitor.l1.latencyMs.toFixed(0)} ms`
        : friendlyTransportLabel(snapshot.transport),
    },
    {
      label: i18nStore.t('machines.machineHealthPanel.stats.cpu'),
      value:
        snapshot.cpuUsagePercent === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : `${snapshot.cpuUsagePercent.toFixed(1)}%`,
      meta:
        snapshot.cpuCores === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noCoreCount')
          : i18nStore.t('machines.machineHealthPanel.dynamic.cores', {
              count: snapshot.cpuCores.toFixed(0),
            }),
    },
    {
      label: i18nStore.t('machines.shared.memory'),
      value:
        snapshot.memoryUsedGB === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
      meta:
        snapshot.memoryAvailableGB === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noFreeMemoryData')
          : i18nStore.t('machines.machineHealthPanel.dynamic.freeMemory', {
              count: snapshot.memoryAvailableGB.toFixed(1),
            }),
    },
    {
      label: i18nStore.t('machines.shared.disk'),
      value:
        snapshot.diskAvailableGB === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : i18nStore.t('machines.machineRowCard.resources.freeGb', {
              free: snapshot.diskAvailableGB.toFixed(1),
            }),
      meta:
        snapshot.agentDispatchable === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.agentDispatchUnknown')
          : snapshot.agentDispatchable
            ? i18nStore.t('machines.machineHealthPanel.dynamic.atLeastOneRuntimeReady')
            : i18nStore.t('machines.machineHealthPanel.dynamic.noRuntimeDispatchable'),
    },
  ]
}

export function buildLevelCards(snapshot: MachineSnapshot): HealthLevelCard[] {
  const readyRuntimeCount = snapshot.agentEnvironment.filter((runtime) => runtime.ready).length

  return [
    {
      id: 'l1',
      label: i18nStore.t('machines.machineHealthPanel.levels.l1Reachability'),
      state: snapshot.monitor.l1?.error
        ? 'error'
        : snapshot.monitor.l1?.reachable === true
          ? 'ok'
          : 'unknown',
      value:
        snapshot.monitor.l1?.reachable === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noReachabilitySample')
          : snapshot.monitor.l1.reachable
            ? i18nStore.t('machines.machineHealthPanel.dynamic.machineReachable')
            : i18nStore.t('machines.machineHealthPanel.dynamic.machineUnreachable'),
      meta: checkedAtLabel(snapshot.monitor.l1?.checkedAt),
    },
    {
      id: 'l2',
      label: i18nStore.t('machines.machineHealthPanel.levels.l2System'),
      state: snapshot.monitor.l2?.error
        ? 'error'
        : snapshot.monitor.l2?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.cpuUsagePercent === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noSystemSnapshot')
          : `CPU ${snapshot.cpuUsagePercent.toFixed(1)}% · RAM ${snapshot.memoryAvailableGB?.toFixed(1) ?? '?'} GB free`,
      meta: checkedAtLabel(snapshot.monitor.l2?.checkedAt),
    },
    {
      id: 'l3',
      label: i18nStore.t('machines.machineHealthPanel.levels.l3Gpu'),
      state: snapshot.monitor.l3?.error
        ? 'error'
        : snapshot.monitor.l3?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.monitor.l3?.available === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noGpuProbe')
          : snapshot.monitor.l3.available
            ? i18nStore.t('machines.machineHealthPanel.dynamic.gpuDetected', {
                count: snapshot.gpus.length,
              })
            : i18nStore.t('machines.machineHealthPanel.dynamic.noGpuDetected'),
      meta: checkedAtLabel(snapshot.monitor.l3?.checkedAt),
    },
    {
      id: 'l4',
      label: i18nStore.t('machines.machineHealthPanel.levels.l4RuntimeEnvironment'),
      state: snapshot.monitor.l4?.error
        ? 'error'
        : snapshot.monitor.l4?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.agentEnvironment.length === 0
          ? i18nStore.t('machines.machineHealthPanel.dynamic.noRuntimeSnapshot')
          : i18nStore.t('machines.machineHealthPanel.dynamic.runtimesReady', {
              ready: readyRuntimeCount,
              total: snapshot.agentEnvironment.length,
            }),
      meta: checkedAtLabel(snapshot.monitor.l4?.checkedAt),
    },
    {
      id: 'l5',
      label: i18nStore.t('machines.machineHealthPanel.levels.l5ToolingAudit'),
      state: snapshot.monitor.l5?.error
        ? 'error'
        : snapshot.monitor.l5?.checkedAt
          ? 'ok'
          : 'unknown',
      value: snapshot.fullAudit?.checkedAt
        ? i18nStore.t('machines.machineHealthPanel.dynamic.auditCaptured')
        : i18nStore.t('machines.machineHealthPanel.dynamic.noToolingAudit'),
      meta: checkedAtLabel(snapshot.monitor.l5?.checkedAt),
    },
  ]
}

export function buildAuditRows(snapshot: MachineSnapshot): HealthAuditRow[] {
  if (!snapshot.fullAudit) {
    return []
  }

  const gitIdentity =
    [snapshot.fullAudit.git?.userName, snapshot.fullAudit.git?.userEmail]
      .filter(Boolean)
      .join(' · ') || null

  const network = snapshot.fullAudit.network

  return [
    {
      kind: 'git',
      label: i18nStore.t('machines.shared.git'),
      installed: toTruthyState(snapshot.fullAudit.git?.installed),
      identity: gitIdentity,
    },
    {
      kind: 'gh-cli',
      label: i18nStore.t('machines.shared.githubCli'),
      installed: toTruthyState(snapshot.fullAudit.ghCLI?.installed),
      authStatus: snapshot.fullAudit.ghCLI?.authStatus ?? null,
    },
    {
      kind: 'network',
      label: i18nStore.t('machines.shared.network'),
      endpoints: [
        { name: 'GitHub', reachable: toTruthyState(network?.githubReachable) },
        { name: 'PyPI', reachable: toTruthyState(network?.pypiReachable) },
        { name: 'npm', reachable: toTruthyState(network?.npmReachable) },
      ],
      auditTimestamp: snapshot.fullAudit.checkedAt ?? null,
    },
  ]
}

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

export function runtimeLabel(runtime: MachineCLIStatus): string {
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

export function levelState(level: { error?: string; checkedAt?: string } | undefined): string {
  if (!level) return 'unknown'
  if (level.error) return 'error'
  if (level.checkedAt) return 'ok'
  return 'unknown'
}

export function stateBadgeVariant(state: string): 'secondary' | 'destructive' | 'outline' {
  switch (state) {
    case 'ok':
      return 'secondary'
    case 'error':
      return 'destructive'
    default:
      return 'outline'
  }
}

export function stateLabel(state: string): string {
  switch (state) {
    case 'ok':
      return 'OK'
    case 'error':
      return 'Error'
    default:
      return i18nStore.t('machines.machineHealthPanel.status.unknown')
  }
}
