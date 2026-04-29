import { i18nStore } from '$lib/i18n/store.svelte'
import { formatMachineRelativeTime } from '../machine-i18n'
import { machineStatusDescription } from '../machine-status'
import type { MachineItem, MachineSnapshot } from '../types'

export type StatusDot = {
  key: string
  label: string
  description: string
  color: 'green' | 'red' | 'amber' | 'gray'
}

export type ResourceBar = {
  key: string
  label: string
  percent: number
  summary: string
  detail: string
  barClass: string
  segments?: { percent: number; barClass: string; label?: string }[]
}

export function buildStatusDots(
  machine: MachineItem,
  snapshot: MachineSnapshot | null,
): StatusDot[] {
  const monitor = snapshot?.monitor

  return [
    {
      key: 'status',
      label: machine.status,
      description: machineStatusDescription(machine.status),
      color: machineStatusColor(machine.status),
    },
    {
      key: 'heartbeat',
      label: machine.last_heartbeat_at
        ? formatMachineRelativeTime(machine.last_heartbeat_at)
        : i18nStore.t('machines.machineRowCard.messages.noHeartbeat'),
      description: i18nStore.t('machines.shared.heartbeat'),
      color: machine.last_heartbeat_at ? 'green' : 'gray',
    },
    {
      key: 'l1',
      label: i18nStore.t('machines.shared.reachability'),
      description:
        monitor?.l1?.reachable === true
          ? i18nStore.t('machines.machineHealthPanel.dynamic.reachable')
          : monitor?.l1?.reachable === false
            ? i18nStore.t('machines.machineHealthPanel.dynamic.unavailable')
            : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet'),
      color: monitorColor(monitor?.l1),
    },
    {
      key: 'l2',
      label: i18nStore.t('machines.shared.system'),
      description:
        snapshot?.cpuUsagePercent !== undefined
          ? `CPU ${snapshot.cpuUsagePercent.toFixed(0)}%`
          : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet'),
      color: monitorColor(monitor?.l2),
    },
    {
      key: 'l3',
      label: 'GPU',
      description: monitor?.l3?.available
        ? i18nStore.t('machines.machineRowCard.resources.gpuSummary', {
            count: snapshot?.gpus.length ?? 0,
          })
        : i18nStore.t('machines.machineHealthPanel.dynamic.noGpuDetected'),
      color: monitorColor(monitor?.l3),
    },
    {
      key: 'l4',
      label: i18nStore.t('machines.shared.runtime'),
      description: snapshot?.agentEnvironment.length
        ? i18nStore.t('machines.machineHealthPanel.dynamic.runtimesReady', {
            ready: snapshot.agentEnvironment.filter((runtime) => runtime.ready).length,
            total: snapshot.agentEnvironment.length,
          })
        : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet'),
      color: monitorColor(monitor?.l4),
    },
    {
      key: 'l5',
      label: i18nStore.t('machines.shared.tooling'),
      description: snapshot?.fullAudit?.checkedAt
        ? i18nStore.t('machines.machineHealthPanel.dynamic.auditCaptured')
        : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet'),
      color: monitorColor(monitor?.l5),
    },
  ]
}

export function buildResourceBars(snapshot: MachineSnapshot | null): ResourceBar[] {
  const gpuAverage =
    snapshot && snapshot.gpus.length > 0
      ? snapshot.gpus.reduce((total, gpu) => total + gpu.utilizationPercent, 0) /
        snapshot.gpus.length
      : undefined

  return [
    {
      key: 'cpu',
      label: 'CPU',
      percent: snapshot?.cpuUsagePercent ?? 0,
      summary:
        snapshot?.cpuUsagePercent === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : `${snapshot.cpuUsagePercent.toFixed(0)}%`,
      detail:
        snapshot?.cpuUsagePercent === undefined
          ? i18nStore.t('machines.machineRowCard.resources.cpuPending')
          : i18nStore.t('machines.machineRowCard.resources.cpuDetail', {
              used: snapshot.cpuUsagePercent.toFixed(1),
              cores: snapshot.cpuCores?.toFixed(0) ?? '?',
            }),
      barClass: toneForPercent(snapshot?.cpuUsagePercent),
    },
    {
      key: 'memory',
      label: i18nStore.t('machines.shared.memory'),
      percent:
        snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
          ? clampPercent((snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100)
          : 0,
      summary:
        snapshot?.memoryUsedGB === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
      detail:
        snapshot?.memoryUsedGB === undefined
          ? i18nStore.t('machines.machineRowCard.resources.memoryPending')
          : i18nStore.t('machines.machineRowCard.resources.memoryDetail', {
              free: snapshot.memoryAvailableGB?.toFixed(1) ?? '?',
              total: snapshot.memoryTotalGB?.toFixed(1) ?? '?',
            }),
      barClass: toneForPercent(
        snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
          ? (snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100
          : undefined,
      ),
    },
    {
      key: 'disk',
      label: i18nStore.t('machines.shared.disk'),
      percent:
        snapshot?.diskTotalGB && snapshot.diskAvailableGB !== undefined
          ? clampPercent(
              ((snapshot.diskTotalGB - snapshot.diskAvailableGB) / snapshot.diskTotalGB) * 100,
            )
          : 0,
      summary:
        snapshot?.diskAvailableGB === undefined
          ? i18nStore.t('machines.machineHealthPanel.dynamic.pending')
          : i18nStore.t('machines.machineRowCard.resources.freeGb', {
              free: snapshot.diskAvailableGB.toFixed(1),
            }),
      detail:
        snapshot?.diskAvailableGB === undefined
          ? i18nStore.t('machines.machineRowCard.resources.diskPending')
          : i18nStore.t('machines.machineRowCard.resources.diskDetail', {
              free: snapshot.diskAvailableGB.toFixed(1),
              total: snapshot.diskTotalGB?.toFixed(1) ?? '?',
            }),
      barClass: toneForPercent(
        snapshot?.diskTotalGB && snapshot.diskAvailableGB !== undefined
          ? ((snapshot.diskTotalGB - snapshot.diskAvailableGB) / snapshot.diskTotalGB) * 100
          : undefined,
      ),
    },
    {
      key: 'gpu',
      label: 'GPU',
      percent: clampPercent(gpuAverage ?? 0),
      summary: snapshot?.gpus.length
        ? i18nStore.t('machines.machineRowCard.resources.gpuSummary', {
            count: snapshot.gpus.length,
          })
        : i18nStore.t('machines.machineHealthPanel.dynamic.noGpuDetected'),
      detail: snapshot?.gpus.length
        ? snapshot.gpus
            .map((gpu) =>
              i18nStore.t('machines.machineRowCard.resources.gpuDetail', {
                name: gpu.name,
                used: gpu.memoryUsedGB.toFixed(1),
                total: gpu.memoryTotalGB.toFixed(1),
                util: gpu.utilizationPercent.toFixed(0),
              }),
            )
            .join('\n')
        : i18nStore.t('machines.machineRowCard.resources.gpuInventoryMissing'),
      barClass: snapshot?.gpuDispatchable ? 'bg-sky-500' : 'bg-slate-400',
      segments: snapshot?.gpus.length
        ? [
            {
              label: 'VRAM',
              percent: clampPercent(
                (snapshot.gpus.reduce((sum, gpu) => sum + gpu.memoryUsedGB, 0) /
                  snapshot.gpus.reduce((sum, gpu) => sum + gpu.memoryTotalGB, 0)) *
                  100,
              ),
              barClass: toneForPercent(
                (snapshot.gpus.reduce((sum, gpu) => sum + gpu.memoryUsedGB, 0) /
                  snapshot.gpus.reduce((sum, gpu) => sum + gpu.memoryTotalGB, 0)) *
                  100,
              ),
            },
            {
              label: i18nStore.t('machines.shared.utilization'),
              percent: clampPercent(gpuAverage ?? 0),
              barClass: snapshot.gpuDispatchable ? 'bg-sky-500' : 'bg-slate-400',
            },
          ]
        : undefined,
    },
  ]
}

function machineStatusColor(status: string): StatusDot['color'] {
  switch (status) {
    case 'online':
      return 'green'
    case 'degraded':
      return 'amber'
    case 'offline':
      return 'red'
    default:
      return 'gray'
  }
}

function monitorColor(
  level: { checkedAt?: string; error?: string } | undefined,
): StatusDot['color'] {
  if (!level?.checkedAt) return 'gray'
  if (level.error) return 'red'
  return 'green'
}

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, value))
}

function toneForPercent(value: number | undefined) {
  if (value === undefined) return 'bg-slate-400'
  if (value >= 85) return 'bg-rose-500'
  if (value >= 65) return 'bg-amber-500'
  return 'bg-emerald-500'
}
