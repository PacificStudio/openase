import { untrack } from 'svelte'
import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import type { ProjectAIFocus } from '$lib/features/chat/project-ai-focus'
import type { ProjectSection } from '$lib/stores/app-context'

type AppPanelContent = { type: 'ticket'; id: string } | { type: 'agent'; id: string }
export type AppTheme = 'light' | 'dark'
type ProjectAssistantFocusRegistration = {
  owner: string
  focus: ProjectAIFocus
  priority: number
  updatedAt: number
}

const flipTheme = (theme: AppTheme): AppTheme => (theme === 'dark' ? 'light' : 'dark')

function createProvisionalOrganization(id: string, previous: Organization | null): Organization {
  return {
    id,
    name: previous?.name ?? 'Loading organization…',
    slug: previous?.slug ?? id,
    default_agent_provider_id: previous?.default_agent_provider_id ?? '',
    status: previous?.status ?? 'active',
  }
}

function createProvisionalProject(orgId: string, id: string, previous: Project | null): Project {
  return {
    id,
    organization_id: previous?.organization_id ?? orgId,
    name: previous?.name ?? 'Loading project…',
    slug: previous?.slug ?? id,
    description: previous?.description ?? '',
    status: previous?.status ?? 'active',
    default_agent_provider_id: previous?.default_agent_provider_id ?? '',
    max_concurrent_agents: previous?.max_concurrent_agents ?? 0,
    agent_run_summary_prompt: previous?.agent_run_summary_prompt ?? '',
    effective_agent_run_summary_prompt: previous?.effective_agent_run_summary_prompt ?? '',
    agent_run_summary_prompt_source: previous?.agent_run_summary_prompt_source ?? 'builtin',
    accessible_machine_ids: previous?.accessible_machine_ids ?? [],
  }
}

function createAppStore() {
  let currentOrg = $state<Organization | null>(null),
    currentProject = $state<Project | null>(null)
  let organizations = $state<Organization[]>([])
  let projects = $state<Project[]>([])
  let providers = $state<AgentProvider[]>([])
  let agentCount = $state(0)
  let currentSection = $state<ProjectSection>('dashboard')
  let appContextLoading = $state(false)
  let appContextError = $state('')
  let appContextKey = $state('')
  let appContextFetchedAt = $state(0)
  let sidebarCollapsed = $state(false),
    mobileSidebarOpen = $state(false),
    newTicketDialogOpen = $state(false),
    rightPanelOpen = $state(false)
  let rightPanelContent = $state<AppPanelContent | null>(null)
  let sseStatus = $state<'idle' | 'connecting' | 'live' | 'retrying'>('idle')
  let theme = $state<AppTheme>('dark')
  let projectAssistantRequest = $state<{ prompt: string; timestamp: number } | null>(null)
  let projectAssistantFocus = $state<ProjectAssistantFocusRegistration[]>([])
  return {
    get currentOrg() {
      return currentOrg
    },
    set currentOrg(v) {
      currentOrg = v
    },
    get currentProject() {
      return currentProject
    },
    set currentProject(v) {
      currentProject = v
    },
    get organizations() {
      return organizations
    },
    set organizations(v) {
      organizations = v
    },
    get projects() {
      return projects
    },
    set projects(v) {
      projects = v
    },
    get providers() {
      return providers
    },
    set providers(v) {
      providers = v
    },
    get agentCount() {
      return agentCount
    },
    set agentCount(v) {
      agentCount = v
    },
    get currentSection() {
      return currentSection
    },
    set currentSection(v) {
      currentSection = v
    },
    get appContextLoading() {
      return appContextLoading
    },
    set appContextLoading(v) {
      appContextLoading = v
    },
    get appContextError() {
      return appContextError
    },
    set appContextError(v) {
      appContextError = v
    },
    get appContextKey() {
      return appContextKey
    },
    set appContextKey(v) {
      appContextKey = v
    },
    get appContextFetchedAt() {
      return appContextFetchedAt
    },
    set appContextFetchedAt(v) {
      appContextFetchedAt = v
    },
    applyAppContext(input: {
      organizations: Organization[]
      projects: Project[]
      providers: AgentProvider[]
      agentCount: number
    }) {
      organizations = input.organizations
      projects = input.projects
      providers = input.providers
      agentCount = input.agentCount
    },
    resolveOrganization(id: string | null) {
      if (!id) {
        return null
      }

      return (
        organizations.find((organization) => organization.id === id) ??
        (currentOrg?.id === id ? currentOrg : createProvisionalOrganization(id, currentOrg))
      )
    },
    resolveProject(orgId: string | null, id: string | null) {
      if (!orgId || !id) {
        return null
      }

      return (
        projects.find((project) => project.id === id) ??
        (currentProject?.id === id
          ? currentProject
          : createProvisionalProject(orgId, id, currentProject))
      )
    },
    get sidebarCollapsed() {
      return sidebarCollapsed
    },
    set sidebarCollapsed(v) {
      sidebarCollapsed = v
    },
    toggleSidebar() {
      sidebarCollapsed = !sidebarCollapsed
    },
    get mobileSidebarOpen() {
      return mobileSidebarOpen
    },
    set mobileSidebarOpen(v) {
      mobileSidebarOpen = v
    },
    openMobileSidebar() {
      mobileSidebarOpen = true
    },
    closeMobileSidebar() {
      mobileSidebarOpen = false
    },
    get newTicketDialogOpen() {
      return newTicketDialogOpen
    },
    set newTicketDialogOpen(v) {
      newTicketDialogOpen = v
    },
    newTicketDefaultStatusId: '' as string,
    openNewTicketDialog(defaultStatusId?: string) {
      this.newTicketDefaultStatusId = defaultStatusId ?? ''
      newTicketDialogOpen = true
    },
    closeNewTicketDialog() {
      newTicketDialogOpen = false
      this.newTicketDefaultStatusId = ''
    },
    get rightPanelOpen() {
      return rightPanelOpen
    },
    openRightPanel(content: AppPanelContent) {
      rightPanelContent = content
      rightPanelOpen = true
    },
    closeRightPanel() {
      rightPanelOpen = false
      rightPanelContent = null
    },
    get rightPanelContent() {
      return rightPanelContent
    },
    get sseStatus() {
      return sseStatus
    },
    set sseStatus(v) {
      sseStatus = v
    },
    get theme() {
      return theme
    },
    setTheme(nextTheme: AppTheme) {
      theme = nextTheme
    },
    toggleTheme() {
      theme = flipTheme(theme)
    },
    get projectAssistantRequest() {
      return projectAssistantRequest
    },
    get projectAssistantFocus() {
      return (
        [...projectAssistantFocus].sort(
          (left, right) => right.priority - left.priority || right.updatedAt - left.updatedAt,
        )[0]?.focus ?? null
      )
    },
    requestProjectAssistant(prompt = '') {
      projectAssistantRequest = { prompt, timestamp: Date.now() }
    },
    consumeProjectAssistantRequest() {
      const request = projectAssistantRequest
      projectAssistantRequest = null
      return request
    },
    setProjectAssistantFocus(owner: string, focus: ProjectAIFocus, priority = 0) {
      const remaining = untrack(() =>
        projectAssistantFocus.filter((registration) => registration.owner !== owner),
      )
      projectAssistantFocus = [...remaining, { owner, focus, priority, updatedAt: Date.now() }]
    },
    clearProjectAssistantFocus(owner: string) {
      const remaining = untrack(() =>
        projectAssistantFocus.filter((registration) => registration.owner !== owner),
      )
      if (remaining.length === untrack(() => projectAssistantFocus.length)) {
        return
      }
      projectAssistantFocus = remaining
    },
  }
}

export const appStore = createAppStore()
