<script lang="ts">
  import { page } from '$app/state'
  import { loadAppContext } from '$lib/api/app-context'
  import { connectEventStream } from '$lib/api/sse'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import { capabilityCatalog } from '$lib/features/capabilities'
  import { GlobalSearchDialog } from '$lib/features/search'
  import { NewTicketDialog } from '$lib/features/tickets'
  import { TicketDrawer } from '$lib/features/ticket-detail'
  import { appStore } from '$lib/stores/app.svelte'
  import type { AppRouteContext, ProjectSection } from '$lib/stores/app-context'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

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
  const projectHealth = $derived.by(() => {
    const status = appStore.currentProject?.status?.toLowerCase()
    if (status === 'healthy' || status === 'active') return 'healthy'
    if (status === 'blocked' || status === 'archived') return 'critical'
    return 'degraded'
  })

  const searchCapability = capabilityCatalog.search
  const newTicketCapability = capabilityCatalog.newTicket
  const isNewTicketEnabled = $derived(
    newTicketCapability.state === 'available' && Boolean(appStore.currentProject?.id),
  )
  const routeKey = $derived(
    `${routeContext.scope}:${routeContext.orgId ?? ''}:${routeContext.scope === 'project' ? routeContext.projectId : ''}`,
  )

  $effect(() => {
    appStore.currentSection = data.currentSection
  })

  $effect(() => {
    const nextOrgId = routeContext.orgId
    const nextProjectId = routeContext.scope === 'project' ? routeContext.projectId : null

    const nextOrg = appStore.resolveOrganization(nextOrgId)
    const nextProject = appStore.resolveProject(nextOrgId, nextProjectId)

    if ((appStore.currentOrg?.id ?? null) !== (nextOrg?.id ?? null)) {
      appStore.currentOrg = nextOrg
    }

    if ((appStore.currentProject?.id ?? null) !== (nextProject?.id ?? null)) {
      appStore.currentProject = nextProject
    }
  })

  $effect(() => {
    let cancelled = false

    const isFresh =
      lastAppContextKey === routeKey &&
      Date.now() - lastAppContextFetchedAt < 30_000 &&
      appStore.organizations.length > 0

    if (isFresh) {
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

        appStore.applyAppContext({
          organizations: payload.organizations,
          projects: payload.projects,
          providers: payload.providers,
          agentCount: payload.agentCount,
        })
        lastAppContextKey = routeKey
        lastAppContextFetchedAt = Date.now()
        appStore.appContextFetchedAt = lastAppContextFetchedAt
        appStore.currentOrg = routeContext.orgId
          ? (payload.organizations.find((organization) => organization.id === routeContext.orgId) ??
            null)
          : null
        appStore.currentProject =
          routeContext.scope === 'project'
            ? (payload.projects.find((project) => project.id === routeContext.projectId) ?? null)
            : null
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

    return connectEventStream(`/api/v1/projects/${projectId}/activity/stream`, {
      onEvent: () => {},
      onStateChange: (state) => {
        appStore.sseStatus = state
      },
      onError: () => {
        appStore.sseStatus = 'retrying'
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
    }

    window.addEventListener('keydown', handleKeydown)
    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
  })

  function handleOpenSearch() {
    searchOpen = true
  }

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
    sseStatus={appStore.sseStatus}
    searchEnabled={searchCapability.state === 'available' && appStore.organizations.length > 0}
    newTicketEnabled={isNewTicketEnabled}
    newTicketTitle={newTicketCapability.summary}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
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
        projectName={appStore.currentProject?.name ?? ''}
        {projectHealth}
        agentCount={appStore.agentCount}
        onToggleCollapse={() => appStore.toggleSidebar()}
      />
    </aside>

    <main class="flex min-w-0 flex-1 flex-col overflow-auto">
      {@render children()}
    </main>
  </div>

  <TicketDrawer
    projectId={appStore.currentProject?.id}
    ticketId={currentTicketId}
    open={appStore.rightPanelOpen}
    onOpenChange={(open) => {
      if (!open) {
        appStore.closeRightPanel()
      }
    }}
  />

  <NewTicketDialog />

  <GlobalSearchDialog
    bind:open={searchOpen}
    organizations={appStore.organizations}
    projects={appStore.projects}
    currentOrg={appStore.currentOrg}
    currentProject={appStore.currentProject}
    currentSection={data.currentSection}
    newTicketEnabled={isNewTicketEnabled}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenTicket={(ticketId) => appStore.openRightPanel({ type: 'ticket', id: ticketId })}
  />
</div>
