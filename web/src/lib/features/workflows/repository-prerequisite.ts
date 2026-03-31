export type WorkflowRepositoryPrerequisite = {
  kind: 'ready'
  repoCount: number
  action: 'none'
}

export function mapWorkflowRepositoryPrerequisite(payload: {
  prerequisite: {
    kind: string
    repo_count: number
    action: string
  }
}): WorkflowRepositoryPrerequisite {
  return {
    kind: 'ready',
    repoCount: payload.prerequisite.repo_count,
    action: 'none',
  }
}
