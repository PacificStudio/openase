import { formatRelativeTime } from '$lib/utils'
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

export type HealthAuditRow = {
  label: string
  value: string
  detail: string
}

export function buildStatCards(snapshot: MachineSnapshot): HealthStatCard[] {
  return [
    {
      label: 'Reachability',
      value:
        snapshot.monitor.l1?.reachable === undefined
          ? 'Unknown'
          : snapshot.monitor.l1.reachable
            ? 'Reachable'
            : 'Unavailable',
      meta: snapshot.monitor.l1?.latencyMs
        ? `${snapshot.monitor.l1.latencyMs.toFixed(0)} ms`
        : (snapshot.transport ?? 'No transport'),
    },
    {
      label: 'CPU',
      value:
        snapshot.cpuUsagePercent === undefined
          ? 'Pending'
          : `${snapshot.cpuUsagePercent.toFixed(1)}%`,
      meta:
        snapshot.cpuCores === undefined ? 'No core count' : `${snapshot.cpuCores.toFixed(0)} cores`,
    },
    {
      label: 'Memory',
      value:
        snapshot.memoryUsedGB === undefined
          ? 'Pending'
          : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
      meta:
        snapshot.memoryAvailableGB === undefined
          ? 'No free memory data'
          : `${snapshot.memoryAvailableGB.toFixed(1)} GB free`,
    },
    {
      label: 'Disk',
      value:
        snapshot.diskAvailableGB === undefined
          ? 'Pending'
          : `${snapshot.diskAvailableGB.toFixed(1)} GB free`,
      meta:
        snapshot.agentDispatchable === undefined
          ? 'Agent dispatch unknown'
          : snapshot.agentDispatchable
            ? 'At least one runtime ready'
            : 'No runtime currently dispatchable',
    },
  ]
}

export function buildLevelCards(snapshot: MachineSnapshot): HealthLevelCard[] {
  const readyRuntimeCount = snapshot.agentEnvironment.filter((runtime) => runtime.ready).length

  return [
    {
      id: 'l1',
      label: 'L1 Reachability',
      state: snapshot.monitor.l1?.error
        ? 'error'
        : snapshot.monitor.l1?.reachable === true
          ? 'ok'
          : 'unknown',
      value:
        snapshot.monitor.l1?.reachable === undefined
          ? 'No reachability sample yet'
          : snapshot.monitor.l1.reachable
            ? 'Machine is reachable'
            : 'Machine is unreachable',
      meta: checkedAtLabel(snapshot.monitor.l1?.checkedAt),
    },
    {
      id: 'l2',
      label: 'L2 System',
      state: snapshot.monitor.l2?.error
        ? 'error'
        : snapshot.monitor.l2?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.cpuUsagePercent === undefined
          ? 'No system resource snapshot yet'
          : `CPU ${snapshot.cpuUsagePercent.toFixed(1)}% · RAM ${snapshot.memoryAvailableGB?.toFixed(1) ?? '?'} GB free`,
      meta: checkedAtLabel(snapshot.monitor.l2?.checkedAt),
    },
    {
      id: 'l3',
      label: 'L3 GPU',
      state: snapshot.monitor.l3?.error
        ? 'error'
        : snapshot.monitor.l3?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.monitor.l3?.available === undefined
          ? 'No GPU probe snapshot yet'
          : snapshot.monitor.l3.available
            ? `${snapshot.gpus.length} GPU detected`
            : 'No GPU detected',
      meta: checkedAtLabel(snapshot.monitor.l3?.checkedAt),
    },
    {
      id: 'l4',
      label: 'L4 Runtime Environment',
      state: snapshot.monitor.l4?.error
        ? 'error'
        : snapshot.monitor.l4?.checkedAt
          ? 'ok'
          : 'unknown',
      value:
        snapshot.agentEnvironment.length === 0
          ? 'No runtime environment snapshot yet'
          : `${readyRuntimeCount}/${snapshot.agentEnvironment.length} runtimes ready`,
      meta: checkedAtLabel(snapshot.monitor.l4?.checkedAt),
    },
    {
      id: 'l5',
      label: 'L5 Tooling Audit',
      state: snapshot.monitor.l5?.error
        ? 'error'
        : snapshot.monitor.l5?.checkedAt
          ? 'ok'
          : 'unknown',
      value: snapshot.fullAudit?.checkedAt
        ? 'Git, GitHub CLI, token, and network audit captured'
        : 'No tooling audit snapshot yet',
      meta: checkedAtLabel(snapshot.monitor.l5?.checkedAt),
    },
  ]
}

export function buildAuditRows(snapshot: MachineSnapshot): HealthAuditRow[] {
  if (!snapshot.fullAudit) {
    return []
  }

  return [
    {
      label: 'Git',
      value: truthyLabel(snapshot.fullAudit.git?.installed),
      detail:
        [snapshot.fullAudit.git?.userName, snapshot.fullAudit.git?.userEmail]
          .filter(Boolean)
          .join(' · ') || 'No git identity recorded',
    },
    {
      label: 'GitHub CLI',
      value: truthyLabel(snapshot.fullAudit.ghCLI?.installed),
      detail: snapshot.fullAudit.ghCLI?.authStatus ?? 'No auth status recorded',
    },
    {
      label: 'GitHub token',
      value: snapshot.fullAudit.githubTokenProbe?.state ?? 'Unknown',
      detail: snapshot.fullAudit.githubTokenProbe?.permissions.length
        ? snapshot.fullAudit.githubTokenProbe.permissions.join(', ')
        : (snapshot.fullAudit.githubTokenProbe?.lastError ?? 'No scopes recorded'),
    },
    {
      label: 'Network',
      value: networkSummary(snapshot),
      detail: snapshot.fullAudit.checkedAt
        ? `Audit captured ${formatRelativeTime(snapshot.fullAudit.checkedAt)}`
        : 'No audit timestamp recorded',
    },
  ]
}

export function checkedAtLabel(value: string | undefined): string {
  return value ? `Checked ${formatRelativeTime(value)}` : 'Not checked yet'
}

export function truthyLabel(value: boolean | undefined): string {
  if (value === undefined) return 'Unknown'
  return value ? 'Yes' : 'No'
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
      return 'Unknown'
  }
}

function networkSummary(snapshot: MachineSnapshot): string {
  const network = snapshot.fullAudit?.network
  if (!network) {
    return 'Unknown'
  }

  return [
    `GitHub ${truthyLabel(network.githubReachable)}`,
    `PyPI ${truthyLabel(network.pypiReachable)}`,
    `npm ${truthyLabel(network.npmReachable)}`,
  ].join(' · ')
}
