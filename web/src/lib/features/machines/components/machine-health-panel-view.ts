import { formatRelativeTime } from '$lib/utils'
import { friendlyTransportLabel } from '../machine-setup'
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
      label: 'Reachability',
      value:
        snapshot.monitor.l1?.reachable === undefined
          ? 'Unknown'
          : snapshot.monitor.l1.reachable
            ? 'Reachable'
            : 'Unavailable',
      meta: snapshot.monitor.l1?.latencyMs
        ? `${snapshot.monitor.l1.latencyMs.toFixed(0)} ms`
        : friendlyTransportLabel(snapshot.transport),
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
        ? 'Git, GitHub CLI observation, and network audit captured'
        : 'No tooling audit snapshot yet',
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
      label: 'Git',
      installed: toTruthyState(snapshot.fullAudit.git?.installed),
      identity: gitIdentity,
    },
    {
      kind: 'gh-cli',
      label: 'GitHub CLI',
      installed: toTruthyState(snapshot.fullAudit.ghCLI?.installed),
      authStatus: snapshot.fullAudit.ghCLI?.authStatus ?? null,
    },
    {
      kind: 'network',
      label: 'Network',
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
  return value ? `Checked ${formatRelativeTime(value)}` : 'Not checked yet'
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
      return 'Unknown'
  }
}
