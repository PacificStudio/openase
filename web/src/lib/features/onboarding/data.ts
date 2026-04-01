import {
  getProject,
  getSecuritySettings,
  listAgents,
  listProjectRepos,
  listProviders,
  listStatuses,
  listTickets,
  listWorkflows,
  listGitHubNamespaces,
} from '$lib/api/openase'
import type { OnboardingData } from './types'
import { parseGitHubTokenState } from './github'

export async function loadOnboardingData(
  projectId: string,
  orgId: string,
): Promise<OnboardingData> {
  const [
    projectPayload,
    securityPayload,
    repoPayload,
    providerPayload,
    agentPayload,
    workflowPayload,
    ticketPayload,
    statusPayload,
  ] = await Promise.all([
    getProject(projectId),
    getSecuritySettings(projectId).catch(() => null),
    listProjectRepos(projectId),
    listProviders(orgId),
    listAgents(projectId),
    listWorkflows(projectId),
    listTickets(projectId),
    listStatuses(projectId),
  ])

  const github = parseGitHubTokenState(securityPayload)

  let namespaces: OnboardingData['repo']['namespaces'] = []
  if (github.probeStatus === 'valid') {
    try {
      const nsPayload = await listGitHubNamespaces(projectId)
      namespaces = nsPayload.namespaces
    } catch {
      // GitHub namespaces not available yet
    }
  }

  return {
    github,
    repo: {
      repos: repoPayload.repos,
      namespaces,
    },
    provider: {
      providers: providerPayload.providers,
      selectedProviderId: projectPayload.project.default_agent_provider_id ?? '',
    },
    agentWorkflow: {
      agents: agentPayload.agents,
      workflows: workflowPayload.workflows,
      statuses: statusPayload.statuses,
    },
    firstTicket: {
      ticketCount: ticketPayload.tickets.length,
    },
    aiDiscovery: {
      completed: false,
    },
    projectStatus: 'Planned',
  }
}
