<script lang="ts">
  import { page } from '$app/state'
  import { loadAppContext } from '$lib/api/app-context'
  import { getProject } from '$lib/api/openase'
  import type { ProjectAIFocus } from '$lib/features/chat'
  import {
    isProjectDashboardRefreshEvent,
    readProjectDashboardRefreshSections,
    retainProjectEventBus,
    subscribeProjectEvents,
  } from '$lib/features/project-events'
  import { authStore } from '$lib/stores/auth.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { type AppRouteContext, type ProjectSection } from '$lib/stores/app-context'
  import type { Snippet } from 'svelte'
  import ProjectShellFrame from './project-shell-frame.svelte'
  import {
    applyLoadedAppContext,
    consumeProjectAssistantRequest,
    logoutProjectShellSession,
    openSettingsForRoute,
    resolveAssistantFocus,
    settingsHrefForRoute,
    syncResolvedRouteContext,
  } from './project-shell-runtime.svelte'
  import { bindProjectShellShortcuts } from './project-shell-shortcuts'
  import {
    getProjectHealth,
    getProjectHealthLabel,
    isAppContextFresh,
    mergeProjectIntoAppContext,
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

        const applied = applyLoadedAppContext(payload, routeContext, routeKey)
        lastAppContextKey = applied.routeKey
        lastAppContextFetchedAt = applied.fetchedAt
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
    if (viewport.isMobile) {
      appStore.closeMobileSidebar()
      projectAssistantOpen = false
    }
    searchOpen = true
  }

  function handleOpenProjectAssistant(initialPrompt = '') {
    projectAssistantPrompt = initialPrompt
    searchOpen = false
    if (viewport.isMobile) {
      appStore.closeMobileSidebar()
    }
    projectAssistantOpen = true
  }

  $effect(() => {
    consumeProjectAssistantRequest(handleOpenProjectAssistant)
  })

  const assistantFocus = $derived<ProjectAIFocus | null>(
    resolveAssistantFocus(appStore.currentProject?.id, appStore.projectAssistantFocus),
  )

  function handleNewTicket() {
    if (viewport.isMobile) {
      appStore.closeMobileSidebar()
      projectAssistantOpen = false
    }
    appStore.openNewTicketDialog()
  }
  function handleToggleTheme() {
    appStore.toggleTheme()
  }

  function handleOpenSettings() {
    openSettingsForRoute(routeContext, 'settings')
  }

  async function handleLogout() {
    if (logoutPending) {
      return
    }

    await logoutProjectShellSession(
      `${page.url.pathname}${page.url.search}${page.url.hash}`,
      (value) => {
        logoutPending = value
      },
    )
  }

  function handleCreateOrg() {
    createOrgOpen = true
  }

  function handleCreateProject() {
    createProjectOpen = true
  }

  function handleLogoutClick() {
    void handleLogout()
  }

  function handleToggleSidebar() {
    appStore.toggleSidebar()
  }

  function handleCloseProjectAssistant() {
    projectAssistantOpen = false
  }

  const settingsEnabled = $derived(routeContext.scope === 'project')
  const settingsHref = $derived(settingsHrefForRoute(routeContext))
  const currentUser = $derived(authStore.user)
</script>

<ProjectShellFrame
  {children}
  {currentPath}
  currentSection={data.currentSection}
  currentOrg={appStore.currentOrg}
  currentProject={appStore.currentProject}
  organizations={appStore.organizations}
  projects={appStore.projects}
  agentCount={appStore.agentCount}
  sseStatus={appStore.sseStatus}
  sidebarCollapsed={appStore.sidebarCollapsed}
  bind:searchOpen
  bind:createOrgOpen
  bind:createProjectOpen
  bind:projectAssistantOpen
  bind:assistantWidth
  bind:resizing
  {currentTicketId}
  {projectAssistantPrompt}
  {assistantFocus}
  projectHealth={appStore.currentProject ? projectHealth : null}
  {projectHealthLabel}
  {isNewTicketEnabled}
  {settingsEnabled}
  {settingsHref}
  userDisplayName={currentUser?.displayName ?? ''}
  userPrimaryEmail={currentUser?.primaryEmail ?? ''}
  userAvatarURL={currentUser?.avatarURL ?? ''}
  {logoutPending}
  onToggleTheme={handleToggleTheme}
  onNewTicket={handleNewTicket}
  onOpenSearch={handleOpenSearch}
  onCreateOrg={handleCreateOrg}
  onCreateProject={handleCreateProject}
  onOpenSettings={handleOpenSettings}
  onLogout={handleLogoutClick}
  onToggleSidebar={handleToggleSidebar}
  onOpenProjectAssistant={handleOpenProjectAssistant}
  onCloseProjectAssistant={handleCloseProjectAssistant}
/>
