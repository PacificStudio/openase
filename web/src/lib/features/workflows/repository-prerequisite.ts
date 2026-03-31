export type WorkflowRepositoryPrerequisite =
  | {
      kind: 'ready'
      repoCount: number
      primaryRepoId: string
      primaryRepoName: string
      action: 'none'
    }
  | {
      kind: 'missing_primary_repo'
      repoCount: number
      action: 'bind_primary_repo'
    }

export function mapWorkflowRepositoryPrerequisite(payload: {
  prerequisite: {
    kind: string
    repo_count: number
    primary_repo_id?: string
    primary_repo_name?: string
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
      action: 'none',
    }
  }

  return {
    kind: 'missing_primary_repo',
    repoCount: prerequisite.repo_count,
    action: 'bind_primary_repo',
  }
}
