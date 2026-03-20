export type Organization = {
  id: string
  name: string
  slug: string
  default_agent_provider_id?: string | null
}

export type ProjectStatus = 'planning' | 'active' | 'paused' | 'archived'

export type Project = {
  id: string
  organization_id: string
  name: string
  slug: string
  description: string
  status: ProjectStatus
  default_workflow_id?: string | null
  default_agent_provider_id?: string | null
  max_concurrent_agents: number
}
