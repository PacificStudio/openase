export type TicketStatusOption = {
  id: string
  name: string
  color: string
}

export type TicketReferenceOption = {
  id: string
  identifier: string
  title: string
}

export type TicketRepoOption = {
  id: string
  name: string
  defaultBranch: string
}

export type TicketDetail = {
  id: string
  identifier: string
  title: string
  description: string
  status: TicketStatusOption
  priority: 'urgent' | 'high' | 'medium' | 'low'
  type: 'feature' | 'bugfix' | 'refactor' | 'chore'
  workflow?: { id: string; name: string; type: string }
  assignedAgent?: { id: string; name: string; provider: string }
  repoScopes: Array<{
    id: string
    repoId: string
    repoName: string
    branchName: string
    prUrl?: string
    prStatus?: string
    ciStatus?: string
    isPrimaryScope: boolean
  }>
  attemptCount: number
  costAmount: number
  budgetUsd: number
  dependencies: Array<{
    id: string
    targetId: string
    identifier: string
    title: string
    relation: string
  }>
  externalLinks: Array<{
    id: string
    type: string
    url: string
    externalId: string
    title?: string
    status?: string
    relation: string
  }>
  children: Array<{ id: string; identifier: string; title: string; status: string }>
  createdBy: string
  createdAt: string
  updatedAt: string
  startedAt?: string
  completedAt?: string
}

export type HookExecution = {
  id: string
  hookName: string
  status: 'pass' | 'fail' | 'running' | 'timeout'
  duration?: number
  output?: string
  timestamp: string
}

export type TicketActivity = {
  id: string
  type: string
  message: string
  timestamp: string
  agentName?: string
}
