import type { Machine } from '$lib/api/contracts'
import type { MachineDraft, MachineDraftParseResult } from './types'
import {
  getWorkspaceRootRecommendation,
  normalizeExecutionMode,
  normalizeReachabilityMode,
} from './machine-guidance'
import {
  coerceExecutionMode,
  parseMachinePort,
  resolveConnectionMode,
  resolveLegacyConnectionMode,
  splitLabels,
  splitLines,
  validateMachineIdentity,
  validateTransportFields,
} from './machine-semantics'
import { normalizeMachineStatus } from './machine-status'

export { parseMachineSnapshot } from './snapshot'
export * from './machine-guidance'
export * from './machine-status'

export function createEmptyMachineDraft(): MachineDraft {
  const draft: MachineDraft = {
    name: '',
    host: '',
    port: '22',
    reachabilityMode: 'direct_connect',
    executionMode: 'websocket',
    sshUser: '',
    sshKeyPath: '',
    advertisedEndpoint: '',
    description: '',
    labels: '',
    status: 'maintenance',
    workspaceRoot: '',
    agentCLIPath: '',
    envVars: '',
  }

  return { ...draft, workspaceRoot: getWorkspaceRootRecommendation({ draft, machine: null }).value }
}

export function machineToDraft(machine: Machine): MachineDraft {
  return {
    name: machine.name,
    host: machine.host,
    port: String(machine.port || 22),
    reachabilityMode: normalizeReachabilityMode(
      machine.reachability_mode,
      machine.host,
      machine.connection_mode,
    ),
    executionMode: normalizeExecutionMode(
      machine.execution_mode,
      machine.host,
      machine.connection_mode,
    ),
    sshUser: machine.ssh_user ?? '',
    sshKeyPath: machine.ssh_key_path ?? '',
    advertisedEndpoint: machine.advertised_endpoint ?? '',
    description: machine.description,
    labels: (machine.labels ?? []).join(', '),
    status: normalizeMachineStatus(machine.status),
    workspaceRoot: machine.workspace_root ?? '',
    agentCLIPath: machine.agent_cli_path ?? '',
    envVars: (machine.env_vars ?? []).join('\n'),
  }
}

export function updateMachineDraft(
  draft: MachineDraft,
  field: keyof MachineDraft,
  value: string,
  machine: Machine | null,
): MachineDraft {
  const previousRecommendation = getWorkspaceRootRecommendation({ draft, machine }).value
  const nextDraft: MachineDraft = { ...draft, [field]: value }

  if (field === 'reachabilityMode' || field === 'executionMode') {
    const reachabilityMode = normalizeReachabilityMode(nextDraft.reachabilityMode, nextDraft.host)
    const executionMode = normalizeExecutionMode(
      nextDraft.executionMode,
      nextDraft.host,
      resolveLegacyConnectionMode(
        nextDraft.reachabilityMode,
        nextDraft.executionMode,
        nextDraft.host,
        normalizeReachabilityMode,
        normalizeExecutionMode,
      ),
    )

    nextDraft.reachabilityMode = reachabilityMode
    nextDraft.executionMode = coerceExecutionMode(
      reachabilityMode,
      executionMode,
      nextDraft.advertisedEndpoint,
    )
    applyModeDefaults(draft, nextDraft)
  }

  if (field !== 'reachabilityMode' && field !== 'executionMode') {
    return maybeRefreshRecommendedWorkspaceRoot(
      draft,
      nextDraft,
      field,
      machine,
      previousRecommendation,
    )
  }

  return maybeRefreshRecommendedWorkspaceRoot(
    draft,
    nextDraft,
    field,
    machine,
    previousRecommendation,
  )
}

export function parseMachineDraft(draft: MachineDraft): MachineDraftParseResult {
  const reachabilityMode = normalizeReachabilityMode(draft.reachabilityMode, draft.host)
  const executionMode = coerceExecutionMode(
    reachabilityMode,
    normalizeExecutionMode(draft.executionMode, draft.host),
    draft.advertisedEndpoint,
  )
  const connectionMode = resolveConnectionMode(reachabilityMode, executionMode)
  const name = reachabilityMode === 'local' ? 'local' : draft.name.trim()
  const host = reachabilityMode === 'local' ? 'local' : draft.host.trim()
  const port = parseMachinePort(draft.port)
  if (port === null) {
    return { ok: false, error: 'Port must be an integer between 1 and 65535.' }
  }

  const identityError = validateMachineIdentity(name, host)
  if (identityError) {
    return { ok: false, error: identityError }
  }

  const transportError = validateTransportFields(draft, reachabilityMode, executionMode)
  if (transportError) {
    return { ok: false, error: transportError }
  }

  const sshUser = draft.sshUser.trim()
  const sshKeyPath = draft.sshKeyPath.trim()
  const advertisedEndpoint = draft.advertisedEndpoint.trim()
  if (connectionMode === 'ws_listener') {
    try {
      const parsed = new URL(advertisedEndpoint)
      if (parsed.protocol !== 'ws:' && parsed.protocol !== 'wss:') {
        return { ok: false, error: 'Advertised endpoint must use ws:// or wss://.' }
      }
      if (!parsed.host.trim()) {
        return { ok: false, error: 'Advertised endpoint must include a host.' }
      }
    } catch {
      return { ok: false, error: 'Advertised endpoint must be a valid websocket URL.' }
    }
  }

  return {
    ok: true,
    value: {
      name,
      host,
      port,
      reachability_mode: reachabilityMode,
      execution_mode: executionMode,
      connection_mode: connectionMode,
      ssh_user: sshUser,
      ssh_key_path: sshKeyPath,
      advertised_endpoint: connectionMode === 'ws_listener' ? advertisedEndpoint : '',
      description: draft.description.trim(),
      labels: splitLabels(draft.labels),
      status: normalizeMachineStatus(draft.status),
      workspace_root: draft.workspaceRoot.trim(),
      agent_cli_path: draft.agentCLIPath.trim(),
      env_vars: splitLines(draft.envVars),
    },
  }
}

function maybeRefreshRecommendedWorkspaceRoot(
  draft: MachineDraft,
  nextDraft: MachineDraft,
  field: keyof MachineDraft,
  machine: Machine | null,
  previousRecommendation: string,
): MachineDraft {
  const followsRecommendation =
    draft.workspaceRoot.trim() === '' || draft.workspaceRoot.trim() === previousRecommendation
  if (
    (field === 'reachabilityMode' || field === 'executionMode' || field === 'sshUser') &&
    followsRecommendation
  ) {
    nextDraft.workspaceRoot = getWorkspaceRootRecommendation({ draft: nextDraft, machine }).value
  }
  return nextDraft
}

function applyModeDefaults(draft: MachineDraft, nextDraft: MachineDraft) {
  if (nextDraft.reachabilityMode === 'local') {
    if (!nextDraft.name.trim() || nextDraft.name.trim().toLowerCase() === 'local') {
      nextDraft.name = 'local'
    }
    nextDraft.host = 'local'
    nextDraft.port = '22'
    nextDraft.executionMode = 'local_process'
    nextDraft.sshUser = ''
    nextDraft.sshKeyPath = ''
    nextDraft.advertisedEndpoint = ''
    return
  }

  if (draft.reachabilityMode === 'local') {
    if (nextDraft.name.trim().toLowerCase() === 'local') nextDraft.name = ''
    if (nextDraft.host.trim().toLowerCase() === 'local') nextDraft.host = ''
  }

  if (nextDraft.reachabilityMode === 'reverse_connect') {
    nextDraft.executionMode = 'websocket'
    nextDraft.advertisedEndpoint = ''
  }
  if (nextDraft.reachabilityMode === 'direct_connect' && nextDraft.executionMode !== 'websocket') {
    nextDraft.advertisedEndpoint = ''
  }
}
