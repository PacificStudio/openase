import { formatRelativeTime } from '$lib/utils'
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
        ? formatRelativeTime(machine.last_heartbeat_at)
        : 'No heartbeat',
      description: 'Heartbeat',
      color: machine.last_heartbeat_at ? 'green' : 'gray',
    },
    {
      key: 'l1',
      label: 'Reachability',
      description:
        monitor?.l1?.reachable === true
          ? 'Reachable'
          : monitor?.l1?.reachable === false
            ? 'Unreachable'
            : 'Not checked',
      color: monitorColor(monitor?.l1),
    },
    {
      key: 'l2',
      label: 'System',
      description:
        snapshot?.cpuUsagePercent !== undefined
          ? `CPU ${snapshot.cpuUsagePercent.toFixed(0)}%`
          : 'Not checked',
      color: monitorColor(monitor?.l2),
    },
    {
      key: 'l3',
      label: 'GPU',
      description: monitor?.l3?.available ? `${snapshot?.gpus.length ?? 0} GPU` : 'No GPU',
      color: monitorColor(monitor?.l3),
    },
    {
      key: 'l4',
      label: 'Runtime',
      description: snapshot?.agentEnvironment.length
        ? `${snapshot.agentEnvironment.filter((runtime) => runtime.ready).length}/${snapshot.agentEnvironment.length} ready`
        : 'Not checked',
      color: monitorColor(monitor?.l4),
    },
    {
      key: 'l5',
      label: 'Tooling',
      description: snapshot?.fullAudit?.checkedAt ? 'Audit captured' : 'Not checked',
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
          ? 'Pending'
          : `${snapshot.cpuUsagePercent.toFixed(0)}%`,
      detail:
        snapshot?.cpuUsagePercent === undefined
          ? 'CPU usage has not been collected yet.'
          : `${snapshot.cpuUsagePercent.toFixed(1)}% CPU in use across ${snapshot.cpuCores?.toFixed(0) ?? '?'} cores.`,
      barClass: toneForPercent(snapshot?.cpuUsagePercent),
    },
    {
      key: 'memory',
      label: 'Memory',
      percent:
        snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
          ? clampPercent((snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100)
          : 0,
      summary:
        snapshot?.memoryUsedGB === undefined
          ? 'Pending'
          : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
      detail:
        snapshot?.memoryUsedGB === undefined
          ? 'Memory usage has not been collected yet.'
          : `${snapshot.memoryAvailableGB?.toFixed(1) ?? '?'} GB free out of ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB total.`,
      barClass: toneForPercent(
        snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
          ? (snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100
          : undefined,
      ),
    },
    {
      key: 'disk',
      label: 'Disk',
      percent:
        snapshot?.diskTotalGB && snapshot.diskAvailableGB !== undefined
          ? clampPercent(
              ((snapshot.diskTotalGB - snapshot.diskAvailableGB) / snapshot.diskTotalGB) * 100,
            )
          : 0,
      summary:
        snapshot?.diskAvailableGB === undefined
          ? 'Pending'
          : `${snapshot.diskAvailableGB.toFixed(1)} GB free`,
      detail:
        snapshot?.diskAvailableGB === undefined
          ? 'Disk usage has not been collected yet.'
          : `${snapshot.diskAvailableGB.toFixed(1)} GB free out of ${snapshot.diskTotalGB?.toFixed(1) ?? '?'} GB.`,
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
        ? `${snapshot.gpus.length} GPU${snapshot.gpus.length === 1 ? '' : 's'}`
        : 'No GPU',
      detail: snapshot?.gpus.length
        ? snapshot.gpus
            .map(
              (gpu) =>
                `${gpu.name}: ${gpu.memoryUsedGB.toFixed(1)}/${gpu.memoryTotalGB.toFixed(1)} GB VRAM, ${gpu.utilizationPercent.toFixed(0)}% util`,
            )
            .join('\n')
        : 'This machine has no GPU inventory in the latest snapshot.',
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
              label: 'Util',
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

function machineStatusDescription(status: string) {
  switch (status) {
    case 'online':
      return 'Machine is reachable and can accept work.'
    case 'degraded':
      return 'Machine is reachable but at least one monitor reported issues.'
    case 'offline':
      return 'Machine is currently unreachable.'
    default:
      return 'Machine status is unknown.'
  }
}
