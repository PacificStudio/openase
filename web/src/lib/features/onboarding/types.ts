import type {
  AgentProvider,
  GitHubRepositoryNamespaceRecord,
  ProjectRepoRecord,
  TicketStatus,
  Workflow,
  Agent,
} from '$lib/api/contracts'

export type OnboardingStepId =
  | 'github_token'
  | 'repo'
  | 'provider'
  | 'agent_workflow'
  | 'first_ticket'
  | 'ai_discovery'

export type OnboardingStepStatus = 'completed' | 'active' | 'locked'

export type OnboardingStep = {
  id: OnboardingStepId
  label: string
  description: string
  status: OnboardingStepStatus
}

export type GitHubTokenState = {
  hasToken: boolean
  probeStatus: 'unknown' | 'testing' | 'valid' | 'invalid'
  login: string
  confirmed: boolean
}

export type RepoState = {
  repos: ProjectRepoRecord[]
  namespaces: GitHubRepositoryNamespaceRecord[]
}

export type ProviderState = {
  providers: AgentProvider[]
  selectedProviderId: string
}

export type AgentWorkflowState = {
  agents: Agent[]
  workflows: Workflow[]
  statuses: TicketStatus[]
}

export type FirstTicketState = {
  ticketCount: number
}

export type OnboardingData = {
  github: GitHubTokenState
  repo: RepoState
  provider: ProviderState
  agentWorkflow: AgentWorkflowState
  firstTicket: FirstTicketState
  aiDiscovery: {
    completed: boolean
  }
  projectStatus: string
}

export type ProjectBootstrapPreset = {
  roleName: string
  roleSlug: string
  workflowType: string
  pickupStatusName: string
  finishStatusName: string
  agentNameSuggestion: string
  exampleTicketTitle: string
}
