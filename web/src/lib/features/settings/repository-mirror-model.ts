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
  local_path?: string
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
  if (draft.mode === 'register_existing' && !localPath) {
    return { ok: false, error: 'Local path is required.' }
  }

  if (localPath && !localPath.startsWith('/')) {
    return { ok: false, error: 'Local path must be absolute.' }
  }

  return {
    ok: true,
    value: {
      machine_id: machineId,
      ...(localPath ? { local_path: localPath } : {}),
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
  if (defaultMirrorMode(repo) === 'prepare') {
    return ''
  }

  const repositoryURL = repo?.repository_url?.trim() ?? ''
  if (repositoryURL.startsWith('/')) {
    return repositoryURL
  }
  return ''
}

export function suggestRepositoryMirrorLocalPath(
  machine: Machine | null | undefined,
  repo: ProjectRepoRecord | null,
  orgSlug: string | null | undefined,
  projectSlug: string | null | undefined,
): string {
  const mirrorRoot = suggestMirrorRoot(machine)
  const repoName = repo?.name?.trim() ?? ''
  const normalizedOrgSlug = orgSlug?.trim() ?? ''
  const normalizedProjectSlug = projectSlug?.trim() ?? ''
  if (!mirrorRoot || !repoName || !normalizedOrgSlug || !normalizedProjectSlug) {
    return ''
  }

  return joinPathSegments(mirrorRoot, normalizedOrgSlug, normalizedProjectSlug, repoName)
}

function suggestMirrorRoot(machine: Machine | null | undefined): string {
  if (!machine) {
    return ''
  }

  const configuredMirrorRoot = machine.mirror_root?.trim() ?? ''
  if (configuredMirrorRoot) {
    return trimTrailingSlash(configuredMirrorRoot)
  }

  if (isLocalMachine(machine)) {
    return '~/.openase/mirrors'
  }

  const workspaceRoot = trimTrailingSlash(machine.workspace_root?.trim() ?? '')
  if (!workspaceRoot.startsWith('/')) {
    return ''
  }
  const parentPath = workspaceRoot.slice(0, workspaceRoot.lastIndexOf('/')) || '/'
  return joinPathSegments(parentPath, 'mirrors')
}

function isLocalMachine(machine: Machine): boolean {
  return (
    machine.host?.trim().toLowerCase() === 'local' || machine.name?.trim().toLowerCase() === 'local'
  )
}

function joinPathSegments(...segments: string[]): string {
  return segments
    .map((segment, index) => {
      const trimmed = segment.trim()
      if (index === 0) {
        return trimTrailingSlash(trimmed)
      }
      return trimmed.replace(/^\/+|\/+$/g, '')
    })
    .filter(Boolean)
    .join('/')
}

function trimTrailingSlash(raw: string): string {
  if (raw === '/') {
    return raw
  }
  return raw.replace(/\/+$/g, '')
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
