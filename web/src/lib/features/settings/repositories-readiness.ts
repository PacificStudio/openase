import type { ProjectRepoRecord } from '$lib/api/contracts'

export type RepositoryMirrorState =
  | 'missing'
  | 'provisioning'
  | 'ready'
  | 'stale'
  | 'syncing'
  | 'error'
  | 'deleting'

export type RepositoryMirrorAction = 'none' | 'prepare_mirror' | 'wait_for_mirror' | 'sync_mirror'

export type RepositoryMirrorProjection = {
  mirrorCount: number
  mirrorState: RepositoryMirrorState
  mirrorMachineId: string | null
  lastSyncedAt: string | null
  lastVerifiedAt: string | null
  lastError: string | null
  action: RepositoryMirrorAction
}

export type RepositoryBindingsReadiness =
  | { kind: 'missing_repo'; repoCount: number; action: 'add_repo' }
  | { kind: 'ready'; repoCount: number }

const knownMirrorStates = new Set<RepositoryMirrorState>([
  'missing',
  'provisioning',
  'ready',
  'stale',
  'syncing',
  'error',
  'deleting',
])

export function projectRepoMirrorProjection(repo: ProjectRepoRecord): RepositoryMirrorProjection {
  const mirrorState = parseMirrorState(repo.mirror_state ?? undefined)

  return {
    mirrorCount: repo.mirror_count ?? 0,
    mirrorState,
    mirrorMachineId: repo.mirror_machine_id ?? null,
    lastSyncedAt: repo.last_synced_at ?? null,
    lastVerifiedAt: repo.last_verified_at ?? null,
    lastError: repo.last_error ?? null,
    action: repositoryMirrorActionForState(mirrorState),
  }
}

export function deriveRepositoryBindingsReadiness(
  repos: ProjectRepoRecord[],
): RepositoryBindingsReadiness {
  if (repos.length === 0) {
    return {
      kind: 'missing_repo',
      repoCount: 0,
      action: 'add_repo',
    }
  }

  return {
    kind: 'ready',
    repoCount: repos.length,
  }
}

export function repositoryMirrorActionForState(
  state: RepositoryMirrorState,
): RepositoryMirrorAction {
  switch (state) {
    case 'provisioning':
    case 'syncing':
    case 'deleting':
      return 'wait_for_mirror'
    case 'stale':
    case 'error':
      return 'sync_mirror'
    case 'missing':
      return 'prepare_mirror'
    default:
      return 'none'
  }
}

export function repositoryMirrorStateLabel(state: RepositoryMirrorState): string {
  switch (state) {
    case 'ready':
      return 'Mirror ready'
    case 'missing':
      return 'Mirror missing'
    case 'stale':
      return 'Mirror stale'
    case 'error':
      return 'Mirror error'
    case 'provisioning':
      return 'Mirror provisioning'
    case 'syncing':
      return 'Mirror syncing'
    case 'deleting':
      return 'Mirror deleting'
    default:
      return 'Mirror unknown'
  }
}

export function repositoryMirrorToneClasses(state: RepositoryMirrorState): string {
  switch (state) {
    case 'ready':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700'
    case 'stale':
    case 'error':
      return 'border-destructive/30 bg-destructive/10 text-destructive'
    default:
      return 'border-amber-500/30 bg-amber-500/10 text-amber-700'
  }
}

export function formatMirrorTimestamp(value: string | null): string | null {
  if (!value) {
    return null
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toISOString().replace('.000Z', ' UTC').replace('T', ' ')
}

function parseMirrorState(value: string | undefined): RepositoryMirrorState {
  if (value && knownMirrorStates.has(value as RepositoryMirrorState)) {
    return value as RepositoryMirrorState
  }

  return 'missing'
}
