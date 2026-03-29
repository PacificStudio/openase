import type { Machine, ProjectRepoRecord } from '$lib/api/contracts'
import {
  repositoryMirrorActionForState,
  type RepositoryMirrorState,
} from './repositories-readiness'

export type RepositoryMirrorMode = 'register_existing' | 'prepare'

export type RepositoryMirrorDraft = {
  machineId: string
  localPath: string
  mode: RepositoryMirrorMode
}

export type RepositoryMirrorMutationInput = {
  machine_id: string
  local_path: string
  mode: RepositoryMirrorMode
}

export type RepositoryMirrorDraftParseResult =
  | { ok: true; value: RepositoryMirrorMutationInput }
  | { ok: false; error: string }

export type MirrorActionContext = {
  buttonLabel: string
  dialogTitle: string
  submitLabel: string
}

export function createRepositoryMirrorDraft(
  machines: Machine[],
  repo: ProjectRepoRecord | null,
): RepositoryMirrorDraft {
  return {
    machineId: selectDefaultMachineId(machines),
    localPath: defaultMirrorLocalPath(repo),
    mode: defaultMirrorMode(repo),
  }
}

export function parseRepositoryMirrorDraft(
  draft: RepositoryMirrorDraft,
): RepositoryMirrorDraftParseResult {
  const machineId = draft.machineId.trim()
  if (!machineId) {
    return { ok: false, error: 'Target machine is required.' }
  }

  const localPath = draft.localPath.trim()
  if (!localPath) {
    return { ok: false, error: 'Local path is required.' }
  }

  if (!localPath.startsWith('/')) {
    return { ok: false, error: 'Local path must be absolute.' }
  }

  return {
    ok: true,
    value: {
      machine_id: machineId,
      local_path: localPath,
      mode: draft.mode,
    },
  }
}

export function mirrorActionContext(repo: ProjectRepoRecord): MirrorActionContext {
  const action = repositoryMirrorActionForState(parseMirrorState(repo.mirror_state))
  switch (action) {
    case 'sync_mirror':
      return {
        buttonLabel: 'Repair mirror',
        dialogTitle: `Repair ${repo.name} mirror`,
        submitLabel: 'Repair mirror',
      }
    case 'wait_for_mirror':
      return {
        buttonLabel: 'Mirror busy',
        dialogTitle: `Mirror in progress for ${repo.name}`,
        submitLabel: 'Update mirror',
      }
    default:
      return {
        buttonLabel: 'Set up mirror',
        dialogTitle: `Set up ${repo.name} mirror`,
        submitLabel: 'Set up mirror',
      }
  }
}

function defaultMirrorMode(repo: ProjectRepoRecord | null): RepositoryMirrorMode {
  if (!repo) {
    return 'register_existing'
  }

  return repositoryMirrorActionForState(parseMirrorState(repo.mirror_state)) === 'sync_mirror'
    ? 'prepare'
    : 'register_existing'
}

function defaultMirrorLocalPath(repo: ProjectRepoRecord | null): string {
  const repositoryURL = repo?.repository_url?.trim() ?? ''
  if (repositoryURL.startsWith('/')) {
    return repositoryURL
  }
  return ''
}

function selectDefaultMachineId(machines: Machine[]): string {
  const onlineMachines = machines.filter((machine) => machine.status === 'online')
  const preferredMachine =
    onlineMachines.find((machine) => machine.name === 'local') ??
    onlineMachines[0] ??
    machines.find((machine) => machine.name === 'local') ??
    machines[0]

  return preferredMachine?.id ?? ''
}

function parseMirrorState(value: string | null | undefined): RepositoryMirrorState {
  switch (value) {
    case 'missing':
    case 'provisioning':
    case 'ready':
    case 'stale':
    case 'syncing':
    case 'error':
    case 'deleting':
      return value
    default:
      return 'missing'
  }
}
