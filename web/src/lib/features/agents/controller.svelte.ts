import {
  createWorkspaceController,
  toErrorMessage,
  type Organization,
  type Project,
  type WorkspaceStartOptions,
} from '$lib/features/workspace'
import { loadAgentProviders } from './api'
import type { AgentProvider } from './types'

export function createAgentsController() {
  const workspace = createWorkspaceController()
  let providers = $state<AgentProvider[]>([])
  let providerBusy = $state(false)
  let providerError = $state('')

  async function start(options: WorkspaceStartOptions = {}) {
    await workspace.start(options)
    await refreshProviders()
  }

  function destroy() {
    workspace.destroy()
  }

  async function refreshProviders() {
    const organizationId = workspace.state.selectedOrgId
    if (!organizationId) {
      providers = []
      providerError = ''
      return
    }

    providerBusy = true
    providerError = ''
    try {
      providers = await loadAgentProviders(organizationId)
    } catch (error) {
      providers = []
      providerError = toErrorMessage(error)
    } finally {
      providerBusy = false
    }
  }

  async function selectOrganization(organization: Organization) {
    await workspace.selectOrganization(organization)
    await refreshProviders()
  }

  async function selectProject(project: Project) {
    await workspace.selectProject(project)
  }

  function selectedAgent() {
    return (
      workspace.dashboard.agents.find((item) => item.id === workspace.dashboard.selectedAgentId) ??
      null
    )
  }

  function providerForSelectedAgent() {
    const agent = selectedAgent()
    if (!agent) {
      return null
    }

    return providers.find((item) => item.id === agent.provider_id) ?? null
  }

  function agentsForProvider(providerId: string) {
    return workspace.dashboard.agents.filter((item) => item.provider_id === providerId).length
  }

  return {
    workspace,
    get providers() {
      return providers
    },
    get providerBusy() {
      return providerBusy
    },
    get providerError() {
      return providerError
    },
    start,
    destroy,
    refreshProviders,
    selectOrganization,
    selectProject,
    selectedAgent,
    providerForSelectedAgent,
    agentsForProvider,
  }
}
