import type {
  MachineCLIStatus,
  MachineGitAuditView,
  MachineGitHubCLIAuditView,
  MachineGitHubTokenProbeView,
  MachineGPUView,
  MachineMonitorLevel,
  MachineNetworkAuditView,
  MachineSnapshot,
  ResourceMap,
} from './types'

export function parseMachineSnapshot(raw: ResourceMap | null | undefined): MachineSnapshot | null {
  if (!raw || Object.keys(raw).length === 0) {
    return null
  }

  const monitor = asObject(raw.monitor)
  const l1 = parseMonitorLevel(asObject(monitor?.l1))
  const l2 = parseMonitorLevel(asObject(monitor?.l2))
  const l3 = parseMonitorLevel(asObject(monitor?.l3))
  const l4 = parseMonitorLevel(asObject(monitor?.l4))
  const l5 = parseMonitorLevel(asObject(monitor?.l5))
  const monitorErrors = [l1?.error, l2?.error, l3?.error, l4?.error, l5?.error].filter(
    (value): value is string => Boolean(value),
  )

  return {
    transport: asString(raw.transport),
    checkedAt: asString(raw.checked_at) ?? asString(raw.collected_at),
    lastSuccess: asBoolean(raw.last_success),
    cpuCores: asNumber(raw.cpu_cores),
    cpuUsagePercent: asNumber(raw.cpu_usage_percent),
    memoryTotalGB: asNumber(raw.memory_total_gb),
    memoryUsedGB: asNumber(raw.memory_used_gb),
    memoryAvailableGB: asNumber(raw.memory_available_gb),
    diskTotalGB: asNumber(raw.disk_total_gb),
    diskAvailableGB: asNumber(raw.disk_available_gb),
    gpuDispatchable: asBoolean(raw.gpu_dispatchable),
    gpus: parseGPUViews(raw.gpu),
    agentDispatchable: asBoolean(raw.agent_dispatchable),
    agentEnvironmentCheckedAt: asString(raw.agent_environment_checked_at),
    agentEnvironment: parseAgentEnvironment(raw.agent_environment),
    monitor: { l1, l2, l3, l4, l5 },
    fullAudit: parseFullAudit(raw.full_audit),
    monitorErrors,
  }
}

function parseMonitorLevel(raw: ResourceMap | null | undefined): MachineMonitorLevel | undefined {
  if (!raw) {
    return undefined
  }

  return {
    checkedAt: asString(raw.checked_at),
    error: asString(raw.error),
    transport: asString(raw.transport),
    reachable: asBoolean(raw.reachable),
    latencyMs: asNumber(raw.latency_ms),
    failureCause: asString(raw.failure_cause),
    consecutiveFailures: asNumber(raw.consecutive_failures),
    memoryLow: asBoolean(raw.memory_low),
    diskLow: asBoolean(raw.disk_low),
    available: asBoolean(raw.available),
    gpuDispatchable: asBoolean(raw.gpu_dispatchable),
    agentDispatchable: asBoolean(raw.agent_dispatchable),
  }
}

function parseGPUViews(raw: unknown): MachineGPUView[] {
  if (!Array.isArray(raw)) {
    return []
  }

  return raw
    .map((item) => {
      const gpu = asObject(item)
      if (!gpu) {
        return null
      }

      const index = asNumber(gpu.index)
      const name = asString(gpu.name)
      const memoryTotalGB = asNumber(gpu.memory_total_gb)
      const memoryUsedGB = asNumber(gpu.memory_used_gb)
      const utilizationPercent = asNumber(gpu.utilization_percent)
      if (
        index === undefined ||
        !name ||
        memoryTotalGB === undefined ||
        memoryUsedGB === undefined ||
        utilizationPercent === undefined
      ) {
        return null
      }

      return {
        index,
        name,
        memoryTotalGB,
        memoryUsedGB,
        utilizationPercent,
      }
    })
    .filter((item): item is MachineGPUView => item !== null)
    .sort((left, right) => left.index - right.index)
}

function parseAgentEnvironment(value: unknown): MachineCLIStatus[] {
  const raw = asObject(value)
  if (!raw) {
    return []
  }

  const orderedNames = ['claude_code', 'codex', 'gemini']
  const names = [
    ...orderedNames.filter((name) => name in raw),
    ...Object.keys(raw)
      .filter((name) => !orderedNames.includes(name))
      .sort(),
  ]

  return names
    .map((name): MachineCLIStatus | null => {
      const cli = asObject(raw[name])
      if (!cli) {
        return null
      }

      return {
        name,
        installed: asBoolean(cli.installed),
        version: asString(cli.version),
        authStatus: asString(cli.auth_status),
        authMode: asString(cli.auth_mode),
        ready: asBoolean(cli.ready),
      }
    })
    .filter((item): item is MachineCLIStatus => item !== null)
}

function parseFullAudit(value: unknown):
  | {
      checkedAt?: string
      git?: MachineGitAuditView
      ghCLI?: MachineGitHubCLIAuditView
      githubTokenProbe?: MachineGitHubTokenProbeView
      network?: MachineNetworkAuditView
    }
  | undefined {
  const raw = asObject(value)
  if (!raw) {
    return undefined
  }

  const git = asObject(raw.git)
  const ghCLI = asObject(raw.gh_cli)
  const githubTokenProbe = asObject(raw.github_token_probe)
  const network = asObject(raw.network)

  return {
    checkedAt: asString(raw.checked_at),
    git: git
      ? {
          installed: asBoolean(git.installed),
          userName: asString(git.user_name),
          userEmail: asString(git.user_email),
        }
      : undefined,
    ghCLI: ghCLI
      ? {
          installed: asBoolean(ghCLI.installed),
          authStatus: asString(ghCLI.auth_status),
        }
      : undefined,
    githubTokenProbe: githubTokenProbe
      ? {
          checkedAt: asString(githubTokenProbe.checked_at),
          state: asString(githubTokenProbe.state),
          configured: asBoolean(githubTokenProbe.configured),
          valid: asBoolean(githubTokenProbe.valid),
          permissions: asStringArray(githubTokenProbe.permissions),
          repoAccess: asString(githubTokenProbe.repo_access),
          lastError: asString(githubTokenProbe.last_error),
        }
      : undefined,
    network: network
      ? {
          githubReachable: asBoolean(network.github_reachable),
          pypiReachable: asBoolean(network.pypi_reachable),
          npmReachable: asBoolean(network.npm_reachable),
        }
      : undefined,
  }
}

function asObject(value: unknown): ResourceMap | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as ResourceMap) : null
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
}
