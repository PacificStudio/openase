import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDraft,
  MachineExecutionGuide,
  MachineExecutionMode,
  MachineModeGuide,
  MachineReachabilityMode,
  WorkspaceRootRecommendation,
  WorkspaceRootState,
} from './types'
import { normalizeDetectedOS } from './machine-detection'
import { i18nStore } from '$lib/i18n/store.svelte'

export * from './machine-detection'

const defaultRemoteWorkspaceRoot = '/srv/openase/workspace'

export function machineModeGuide(mode: MachineReachabilityMode): MachineModeGuide {
  switch (mode) {
    case 'local':
      return {
        mode: 'local',
        label: i18nStore.t('machines.guidance.local.label'),
        summary: i18nStore.t('machines.guidance.local.summary'),
        requiredFields: i18nStore.t('machines.guidance.local.requiredFields'),
        installMethod: i18nStore.t('machines.guidance.local.installMethod'),
        testSemantics: i18nStore.t('machines.guidance.local.testSemantics'),
        commonErrors: i18nStore.t('machines.guidance.local.commonErrors'),
      }
    case 'direct_connect':
      return {
        mode: 'direct_connect',
        label: i18nStore.t('machines.guidance.directConnect.label'),
        summary: i18nStore.t('machines.guidance.directConnect.summary'),
        requiredFields: i18nStore.t('machines.guidance.directConnect.requiredFields'),
        installMethod: i18nStore.t('machines.guidance.directConnect.installMethod'),
        testSemantics: i18nStore.t('machines.guidance.directConnect.testSemantics'),
        commonErrors: i18nStore.t('machines.guidance.directConnect.commonErrors'),
      }
    case 'reverse_connect':
      return {
        mode: 'reverse_connect',
        label: i18nStore.t('machines.guidance.reverseConnect.label'),
        summary: i18nStore.t('machines.guidance.reverseConnect.summary'),
        requiredFields: i18nStore.t('machines.guidance.reverseConnect.requiredFields'),
        installMethod: i18nStore.t('machines.guidance.reverseConnect.installMethod'),
        testSemantics: i18nStore.t('machines.guidance.reverseConnect.testSemantics'),
        commonErrors: i18nStore.t('machines.guidance.reverseConnect.commonErrors'),
      }
  }
}

export function machineExecutionGuide(mode: MachineExecutionMode): MachineExecutionGuide {
  switch (mode) {
    case 'local_process':
      return {
        mode: 'local_process',
        label: i18nStore.t('machines.guidance.execution.localProcess.label'),
        summary: i18nStore.t('machines.guidance.execution.localProcess.summary'),
      }
    case 'websocket':
      return {
        mode: 'websocket',
        label: i18nStore.t('machines.guidance.execution.websocket.label'),
        summary: i18nStore.t('machines.guidance.execution.websocket.summary'),
      }
  }
}

export type WorkspaceRootContext = {
  draft: MachineDraft
  machine: Machine | null
}

export function normalizeConnectionMode(
  _mode: string | null | undefined,
  host: string | null | undefined,
  reachabilityMode?: string | null | undefined,
  _executionMode?: string | null | undefined,
): MachineConnectionMode {
  const normalizedReachability = normalizeReachabilityMode(reachabilityMode, host)

  if (normalizedReachability === 'local') {
    return 'local'
  }
  if (normalizedReachability === 'reverse_connect') {
    return 'ws_reverse'
  }
  return 'ws_listener'
}

export function normalizeReachabilityMode(
  value: string | null | undefined,
  host: string | null | undefined,
): MachineReachabilityMode {
  switch (value) {
    case 'local':
    case 'direct_connect':
    case 'reverse_connect':
      return value
  }
  return (host ?? '').trim().toLowerCase() === 'local' ? 'local' : 'direct_connect'
}

export function normalizeExecutionMode(
  value: string | null | undefined,
  host: string | null | undefined,
): MachineExecutionMode {
  switch (value) {
    case 'local_process':
    case 'websocket':
      return value
  }
  return (host ?? '').trim().toLowerCase() === 'local' ? 'local_process' : 'websocket'
}

export function machineReachabilityLabel(mode: string | null | undefined): string {
  return machineModeGuide(normalizeReachabilityMode(mode, null)).label
}

export function machineExecutionModeLabel(mode: string | null | undefined): string {
  return machineExecutionGuide(normalizeExecutionMode(mode, null)).label
}

export function machineDetectionMessage(machine: Machine | null, draft?: MachineDraft): string {
  if (machine?.detection_message?.trim()) {
    return machine.detection_message
  }

  const reachabilityMode = normalizeReachabilityMode(
    draft?.reachabilityMode ?? machine?.reachability_mode,
    draft?.host ?? machine?.host,
  )
  if (reachabilityMode === 'local') {
    return i18nStore.t('machines.guidance.detection.local')
  }
  return i18nStore.t('machines.guidance.detection.remote')
}

export function getWorkspaceRootRecommendation(
  input: WorkspaceRootContext,
): WorkspaceRootRecommendation {
  const reachabilityMode = normalizeReachabilityMode(
    input.draft.reachabilityMode || input.machine?.reachability_mode,
    input.draft.host || input.machine?.host,
  )
  const detectedOS = normalizeDetectedOS(input.machine?.detected_os)
  const sshUser = input.draft.sshUser.trim() || input.machine?.ssh_user || 'openase'

  if (reachabilityMode === 'local') {
    return {
      value: '~/.openase/workspace',
      reason: i18nStore.t('machines.guidance.recommendation.local'),
    }
  }
  if (detectedOS === 'darwin') {
    return {
      value: `/Users/${sshUser}/.openase/workspace`,
      reason: i18nStore.t('machines.guidance.recommendation.darwin'),
    }
  }
  if (detectedOS === 'linux') {
    return {
      value: `/home/${sshUser}/.openase/workspace`,
      reason: i18nStore.t('machines.guidance.recommendation.linux'),
    }
  }

  return {
    value: defaultRemoteWorkspaceRoot,
    reason: i18nStore.t('machines.guidance.recommendation.fallback'),
  }
}

export function getWorkspaceRootState(input: WorkspaceRootContext): WorkspaceRootState {
  const recommended = getWorkspaceRootRecommendation(input).value
  const currentValue = input.draft.workspaceRoot.trim()
  const savedValue = input.machine?.workspace_root?.trim() ?? ''

  if (!currentValue) {
    return { kind: 'empty', label: i18nStore.t('machines.guidance.workspaceRoot.empty') }
  }
  if (currentValue === recommended) {
    return {
      kind: 'recommended',
      label: i18nStore.t('machines.guidance.workspaceRoot.recommended'),
    }
  }
  if (savedValue && currentValue === savedValue) {
    return { kind: 'saved', label: i18nStore.t('machines.guidance.workspaceRoot.saved') }
  }
  return { kind: 'manual', label: i18nStore.t('machines.guidance.workspaceRoot.manual') }
}
