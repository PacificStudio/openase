<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import { logoutHumanSession, normalizeReturnTo } from '$lib/api/auth'
  import { loadAppContext } from '$lib/api/app-context'
  import { ApiError } from '$lib/api/client'
  import { getProject } from '$lib/api/openase'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import type { ProjectAIFocus } from '$lib/features/chat'
  import {
    isProjectDashboardRefreshEvent,
    readProjectDashboardRefreshSections,
    retainProjectEventBus,
    subscribeProjectEvents,
  } from '$lib/features/project-events'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    organizationPath,
    projectPath,
    type AppRouteContext,
    type ProjectSection,
  } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'
  import ProjectShellProjectAssistant from './project-shell-project-assistant.svelte'
  import { bindProjectShellShortcuts } from './project-shell-shortcuts'
  import ProjectShellOverlays from './project-shell-overlays.svelte'
  import {
    getProjectHealth,
    getProjectHealthLabel,
    isAppContextFresh,
    mergeProjectIntoAppContext,
    replaceSelectionIfChanged,
    selectCurrentOrg,
    selectCurrentProject,
  } from './project-shell-state'

  type ShellData = { routeContext: AppRouteContext; currentSection: ProjectSection }

  let { children, data }: { children: Snippet; data: ShellData } = $props()

  let currentPath = $derived(page.url.pathname)
  let routeContext = $derived(data.routeContext)
  let lastAppContextKey = ''
  let lastAppContextFetchedAt = 0
  let currentTicketId = $derived(
    appStore.rightPanelContent?.type === 'ticket' ? appStore.rightPanelContent.id : null,
  )
  let searchOpen = $state(false)
  let createOrgOpen = $state(false)
  let createProjectOpen = $state(false)
  let projectAssistantOpen = $state(false)
  let projectAssistantPrompt = $state('')
  let logoutPending = $state(false)

  const DEFAULT_ASSISTANT_WIDTH = 380
  let assistantWidth = $state(DEFAULT_ASSISTANT_WIDTH)
  let resizing = $state(false)
  const projectHealth = $derived.by(() => getProjectHealth(appStore.currentProject))

  const projectHealthLabel = $derived.by(() =>
    getProjectHealthLabel(appStore.currentProject, projectHealth),
  )

  const isNewTicketEnabled = $derived(Boolean(appStore.currentProject?.id))
  const routeKey = $derived(
    `${routeContext.scope}:${routeContext.orgId ?? ''}:${routeContext.scope === 'project' ? routeContext.projectId : ''}`,
  )

  function syncResolvedRouteContext(nextRouteContext: AppRouteContext) {
    const nextOrgId = nextRouteContext.orgId
    const nextProjectId = nextRouteContext.scope === 'project' ? nextRouteContext.projectId : null
    const nextOrg = appStore.resolveOrganization(nextOrgId)
    const nextProject = appStore.resolveProject(nextOrgId, nextProjectId)

    replaceSelectionIfChanged(appStore.currentOrg, nextOrg, (value) => {
      appStore.currentOrg = value
    })
    replaceSelectionIfChanged(appStore.currentProject, nextProject, (value) => {
      appStore.currentProject = value
    })
  }

  function applyLoadedAppContext(
    payload: Awaited<ReturnType<typeof loadAppContext>>,
    nextRouteKey: string,
  ) {
    appStore.applyAppContext({
      organizations: payload.organizations,
      projects: payload.projects,
      providers: payload.providers,
      agentCount: payload.agentCount,
    })
    lastAppContextKey = nextRouteKey
    lastAppContextFetchedAt = Date.now()
    appStore.appContextFetchedAt = lastAppContextFetchedAt
    appStore.currentOrg = selectCurrentOrg(payload, routeContext)
    appStore.currentProject = selectCurrentProject(payload, routeContext)
  }

  $effect(() => {
    appStore.currentSection = data.currentSection
  })

  $effect(() => {
    syncResolvedRouteContext(routeContext)
  })

  $effect(() => {
    let cancelled = false

    if (isAppContextFresh(lastAppContextKey, routeKey, lastAppContextFetchedAt)) {
      return
    }

    appStore.appContextKey = routeKey
    appStore.appContextLoading = true
    appStore.appContextError = ''

    const load = async () => {
      try {
        const payload = await loadAppContext(globalThis.fetch.bind(globalThis), {
          orgId: routeContext.orgId,
          projectId: routeContext.scope === 'project' ? routeContext.projectId : null,
        })
        if (cancelled) return

        applyLoadedAppContext(payload, routeKey)
      } catch (caughtError) {
        if (cancelled) return
        appStore.appContextError =
          caughtError instanceof Error ? caughtError.message : 'Failed to refresh app context.'
      } finally {
        if (!cancelled) {
          appStore.appContextLoading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      appStore.sseStatus = 'idle'
      return
    }

    return retainProjectEventBus(projectId, {
      onStateChange: (state) => {
        appStore.sseStatus = state
      },
    })
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      return
    }

    let cancelled = false
    let refreshInFlight = false
    let queuedRefresh = false

    const refreshProject = async () => {
      if (refreshInFlight) {
        queuedRefresh = true
        return
      }

      refreshInFlight = true
      try {
        do {
          queuedRefresh = false
          const payload = await getProject(projectId)
          if (cancelled || appStore.currentProject?.id !== projectId) {
            return
          }

          const merged = mergeProjectIntoAppContext(
            appStore.projects,
            appStore.currentProject,
            payload.project,
          )
          appStore.projects = merged.projects
          appStore.currentProject = merged.currentProject
        } while (queuedRefresh && !cancelled)
      } catch (caughtError) {
        if (!cancelled) {
          console.error('Failed to refresh project after passive project event:', caughtError)
        }
      } finally {
        refreshInFlight = false
      }
    }

    return subscribeProjectEvents(projectId, (event) => {
      if (!isProjectDashboardRefreshEvent(event)) {
        return
      }
      if (!readProjectDashboardRefreshSections(event).includes('project')) {
        return
      }
      void refreshProject()
    })
  })

  $effect(() => {
    return bindProjectShellShortcuts({
      getProjectAssistantOpen: () => projectAssistantOpen,
      setProjectAssistantOpen: (value) => {
        projectAssistantOpen = value
      },
      openSearch: () => {
        searchOpen = true
      },
      openProjectAssistant: () => {
        handleOpenProjectAssistant()
      },
    })
  })

  function handleOpenSearch() {
    searchOpen = true
  }

  function handleOpenProjectAssistant(initialPrompt = '') {
    projectAssistantPrompt = initialPrompt
    searchOpen = false
    projectAssistantOpen = true
  }

  // Listen for store-based project assistant requests (from onboarding, etc.)
  $effect(() => {
    const request = appStore.projectAssistantRequest
    if (request) {
      const consumed = appStore.consumeProjectAssistantRequest()
      if (consumed) {
        handleOpenProjectAssistant(consumed.prompt)
      }
    }
  })

  const assistantFocus = $derived<ProjectAIFocus | null>(
    appStore.currentProject?.id
      ? appStore.projectAssistantFocus?.projectId === appStore.currentProject.id
        ? appStore.projectAssistantFocus
        : null
      : null,
  )

  function handleNewTicket() {
    appStore.openNewTicketDialog()
  }
  function handleToggleTheme() {
    appStore.toggleTheme()
  }

  function handleOpenSettings() {
    if (routeContext.scope === 'project') {
      void goto(projectPath(routeContext.orgId, routeContext.projectId, 'settings'))
      return
    }
    if (routeContext.scope === 'org') {
      void goto(organizationPath(routeContext.orgId))
      return
    }
    void goto('/')
  }

  async function handleLogout() {
    if (logoutPending) {
      return
    }

    logoutPending = true
    const returnTo = normalizeReturnTo(`${page.url.pathname}${page.url.search}${page.url.hash}`)
    const authMode = authStore.authMode

    try {
      await logoutHumanSession()
    } catch (caughtError) {
      if (!(caughtError instanceof ApiError) || caughtError.status !== 401) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to log out.',
        )
        logoutPending = false
        return
      }
    }

    authStore.clear()
    logoutPending = false
    await goto(authMode === 'oidc' ? `/login?return_to=${encodeURIComponent(returnTo)}` : '/')
  }
</script>

<div class="bg-background flex h-screen flex-col overflow-hidden">
  <TopBar
    organizations={appStore.organizations}
    projects={appStore.projects}
    currentOrgId={appStore.currentOrg?.id ?? null}
    currentProjectId={appStore.currentProject?.id ?? null}
    currentSection={data.currentSection}
    orgName={appStore.currentOrg?.name ?? 'No organization'}
    projectName={appStore.currentProject?.name ?? ''}
    projectHealth={appStore.currentProject ? projectHealth : null}
    {projectHealthLabel}
    sseStatus={appStore.sseStatus}
    searchEnabled={appStore.organizations.length > 0}
    newTicketEnabled={isNewTicketEnabled}
    settingsEnabled={routeContext.scope === 'project'}
    settingsHref={routeContext.scope === 'project'
      ? projectPath(routeContext.orgId, routeContext.projectId, 'settings')
      : ''}
    userDisplayName={authStore.user?.displayName ?? ''}
    userPrimaryEmail={authStore.user?.primaryEmail ?? ''}
    userAvatarURL={authStore.user?.avatarURL ?? ''}
    {logoutPending}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
    onCreateOrg={() => {
      createOrgOpen = true
    }}
    onCreateProject={() => {
      createProjectOpen = true
    }}
    onOpenSettings={handleOpenSettings}
    onLogout={() => {
      void handleLogout()
    }}
  />

  <div class="flex flex-1 overflow-hidden">
    <aside
      class={cn(
        'border-border bg-sidebar flex h-full flex-col border-r transition-[width] duration-200 ease-in-out',
        appStore.sidebarCollapsed ? 'w-[52px]' : 'w-[240px]',
      )}
    >
      <Sidebar
        collapsed={appStore.sidebarCollapsed}
        {currentPath}
        currentOrgId={appStore.currentOrg?.id ?? null}
        currentProjectId={appStore.currentProject?.id ?? null}
        projectSelected={Boolean(appStore.currentProject)}
        agentCount={appStore.agentCount}
        onOpenProjectAssistant={() => handleOpenProjectAssistant()}
        onToggleCollapse={() => appStore.toggleSidebar()}
      />
    </aside>

    <main class={cn('flex min-w-0 flex-1 flex-col overflow-auto', resizing && 'select-none')}>
      {@render children()}
    </main>

    {#if appStore.currentOrg?.id && appStore.currentProject?.id}
      <ProjectShellProjectAssistant
        organizationId={appStore.currentOrg.id}
        projectId={appStore.currentProject.id}
        defaultProviderId={appStore.currentProject.default_agent_provider_id ?? null}
        focus={assistantFocus}
        open={projectAssistantOpen}
        initialPrompt={projectAssistantPrompt}
        bind:width={assistantWidth}
        bind:resizing
        onClose={() => {
          projectAssistantOpen = false
        }}
      />
    {/if}
  </div>

  <ProjectShellOverlays
    currentOrg={appStore.currentOrg}
    currentProject={appStore.currentProject}
    currentSection={data.currentSection}
    {currentTicketId}
    bind:searchOpen
    bind:createOrgOpen
    bind:createProjectOpen
    newTicketEnabled={isNewTicketEnabled}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenProjectAssistant={handleOpenProjectAssistant}
  />
</div>
