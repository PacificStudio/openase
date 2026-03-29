export type WorkflowRepositoryPrerequisite =
  | {
      kind: 'ready'
      repoCount: number
      primaryRepoId: string
      primaryRepoName: string
      mirrorCount: number
      mirrorState: 'ready'
      action: 'none'
    }
  | {
      kind: 'missing_primary_repo'
      repoCount: number
      action: 'bind_primary_repo'
    }
  | {
      kind: 'primary_mirror_not_ready'
      repoCount: number
      primaryRepoId: string
      primaryRepoName: string
      mirrorCount: number
      mirrorState: 'missing' | 'provisioning' | 'ready' | 'stale' | 'syncing' | 'error' | 'deleting'
      mirrorMachineId: string | null
      mirrorLastError: string | null
      action: 'prepare_primary_mirror' | 'wait_for_primary_mirror' | 'sync_primary_mirror'
    }

type WorkflowMirrorState = Extract<
  WorkflowRepositoryPrerequisite,
  { kind: 'primary_mirror_not_ready' }
>['mirrorState']

type WorkflowMirrorAction = Extract<
  WorkflowRepositoryPrerequisite,
  { kind: 'primary_mirror_not_ready' }
>['action']

export function mapWorkflowRepositoryPrerequisite(payload: {
  prerequisite: {
    kind: string
    repo_count: number
    primary_repo_id?: string
    primary_repo_name?: string
    mirror_count: number
    mirror_state?: string
    mirror_machine_id?: string
    mirror_last_error?: string
    action: string
  }
}): WorkflowRepositoryPrerequisite {
  const prerequisite = payload.prerequisite

  if (prerequisite.kind === 'missing_primary_repo') {
    return {
      kind: 'missing_primary_repo',
      repoCount: prerequisite.repo_count,
      action: 'bind_primary_repo',
    }
  }

  if (prerequisite.kind === 'ready') {
    return {
      kind: 'ready',
      repoCount: prerequisite.repo_count,
      primaryRepoId: prerequisite.primary_repo_id ?? '',
      primaryRepoName: prerequisite.primary_repo_name ?? '',
      mirrorCount: prerequisite.mirror_count,
      mirrorState: 'ready',
      action: 'none',
    }
  }

  return {
    kind: 'primary_mirror_not_ready',
    repoCount: prerequisite.repo_count,
    primaryRepoId: prerequisite.primary_repo_id ?? '',
    primaryRepoName: prerequisite.primary_repo_name ?? '',
    mirrorCount: prerequisite.mirror_count,
    mirrorState: (prerequisite.mirror_state ?? 'missing') as WorkflowMirrorState,
    mirrorMachineId: prerequisite.mirror_machine_id ?? null,
    mirrorLastError: prerequisite.mirror_last_error ?? null,
    action: (prerequisite.action ?? 'prepare_primary_mirror') as WorkflowMirrorAction,
  }
}
