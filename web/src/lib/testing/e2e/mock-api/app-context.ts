import type { MockState } from './constants'
import { ORG_ID, PROJECT_ID } from './constants'
import { clone } from './helpers'

export function buildAppContextPayload(state: MockState, url: URL) {
  const orgId = url.searchParams.get('org_id')
  const projectId = url.searchParams.get('project_id')

  return {
    organizations: clone(state.organizations),
    projects:
      orgId === ORG_ID ? clone(state.projects.filter((project) => project.org_id === orgId)) : [],
    providers:
      orgId === ORG_ID
        ? clone(state.providers.filter((provider) => provider.org_id === orgId))
        : [],
    agent_count:
      projectId === PROJECT_ID
        ? state.agents.filter((agent) => agent.project_id === projectId).length
        : 0,
  }
}
