<script lang="ts">
  import { page } from '$app/state'
  import { loadAppContext } from '$lib/api/app-context'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import { retainProjectEventBus } from '$lib/features/project-events'
  import { ProjectConversationPanel } from '$lib/features/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import type { AppRouteContext, ProjectSection } from '$lib/stores/app-context'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'
  import ProjectShellOverlays from './project-shell-overlays.svelte'
  import {
    getProjectHealth,
    getProjectHealthLabel,
    isAppContextFresh,
    replaceSelectionIfChanged,
    selectCurrentOrg,
    selectCurrentProject,
  } from './project-shell-state'

  type ShellData = {
    routeContext: AppRouteContext
    currentSection: ProjectSection
  }

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

  const DEFAULT_ASSISTANT_WIDTH = 380
  const MIN_ASSISTANT_WIDTH = 280
  const MAX_ASSISTANT_WIDTH = 640
  let assistantWidth = $state(DEFAULT_ASSISTANT_WIDTH)
  let resizing = $state(false)

  function handleResizeStart(event: PointerEvent) {
    event.preventDefault()
    resizing = true
    const startX = event.clientX
    const startWidth = assistantWidth

    function onMove(moveEvent: PointerEvent) {
      const delta = startX - moveEvent.clientX
      assistantWidth = Math.min(
        MAX_ASSISTANT_WIDTH,
        Math.max(MIN_ASSISTANT_WIDTH, startWidth + delta),
      )
    }

    function onUp() {
      resizing = false
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }
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
    const handleKeydown = (event: KeyboardEvent) => {
      if (event.defaultPrevented) {
        return
      }

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        searchOpen = true
      }

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'i') {
        event.preventDefault()
        if (projectAssistantOpen) {
          projectAssistantOpen = false
        } else {
          handleOpenProjectAssistant()
        }
      }
    }

    window.addEventListener('keydown', handleKeydown)
    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
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

  const assistantFocus = $derived(
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
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
    onCreateOrg={() => {
      createOrgOpen = true
    }}
    onCreateProject={() => {
      createProjectOpen = true
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

    {#if projectAssistantOpen && appStore.currentOrg?.id && appStore.currentProject?.id}
      <aside
        class="bg-background relative flex h-full shrink-0 flex-col"
        style="width: {assistantWidth}px"
      >
        <!-- resize handle -->
        <div
          class={cn(
            'absolute inset-y-0 left-0 z-20 w-1 cursor-col-resize transition-colors',
            resizing ? 'bg-primary' : 'bg-border hover:bg-primary/50',
          )}
          role="separator"
          aria-orientation="vertical"
          onpointerdown={handleResizeStart}
        ></div>
        <div class="flex h-full min-w-0 flex-col pl-1">
          <ProjectConversationPanel
            organizationId={appStore.currentOrg.id}
            defaultProviderId={appStore.currentProject.default_agent_provider_id ?? null}
            context={{ projectId: appStore.currentProject.id }}
            focus={assistantFocus}
            title="Project AI"
            placeholder="Ask anything about this project…"
            initialPrompt={projectAssistantPrompt}
            onClose={() => {
              projectAssistantOpen = false
            }}
          />
        </div>
      </aside>
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
