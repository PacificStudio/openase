import {
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

export async function loadOnboardingData(
  projectId: string,
  orgId: string,
): Promise<OnboardingData> {
  const [
    securityPayload,
    repoPayload,
    providerPayload,
    agentPayload,
    workflowPayload,
    ticketPayload,
    statusPayload,
  ] = await Promise.all([
    getSecuritySettings(projectId).catch(() => null),
    listProjectRepos(projectId),
    listProviders(orgId),
    listAgents(projectId),
    listWorkflows(projectId),
    listTickets(projectId),
    listStatuses(projectId),
  ])

  const ghCred = securityPayload as Record<string, unknown> | null
  const ghOutbound = ghCred?.github_outbound_credential as
    | { token_present?: boolean; probe_status?: string; login?: string }
    | null
    | undefined

  let namespaces: OnboardingData['repo']['namespaces'] = []
  if (ghOutbound?.probe_status === 'valid') {
    try {
      const nsPayload = await listGitHubNamespaces(projectId)
      namespaces = nsPayload.namespaces
    } catch {
      // GitHub namespaces not available yet
    }
  }

  return {
    github: {
      hasToken: ghOutbound?.token_present ?? false,
      probeStatus:
        (ghOutbound?.probe_status as OnboardingData['github']['probeStatus']) ?? 'unknown',
      login: (ghOutbound?.login as string) ?? '',
      confirmed: ghOutbound?.probe_status === 'valid',
    },
    repo: {
      repos: repoPayload.repos,
      namespaces,
    },
    provider: {
      providers: providerPayload.providers,
      selectedProviderId: '',
    },
    agentWorkflow: {
      agents: agentPayload.agents,
      workflows: workflowPayload.workflows,
      statuses: statusPayload.statuses,
    },
    firstTicket: {
      ticketCount: ticketPayload.tickets.length,
    },
    projectStatus: 'Planned',
  }
}
