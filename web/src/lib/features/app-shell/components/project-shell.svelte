<script lang="ts">
  import { page } from '$app/state'
  import type { Organization, Project } from '$lib/api/contracts'
  import { connectEventStream } from '$lib/api/sse'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import { capabilityCatalog } from '$lib/features/capabilities'
  import { GlobalSearchDialog } from '$lib/features/search'
  import { NewTicketDialog } from '$lib/features/tickets'
  import { TicketDrawer } from '$lib/features/ticket-detail'
  import { appStore } from '$lib/stores/app.svelte'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

  type ShellData = {
    organizations: Organization[]
    projects: Project[]
    currentOrg: Organization | null
    currentProject: Project | null
    agentCount: number
    currentSection: import('$lib/stores/app-context').ProjectSection
  }

  let { children, data }: { children: Snippet; data: ShellData } = $props()

  let currentPath = $derived(page.url.pathname)
  let currentTicketId = $derived(
    appStore.rightPanelContent?.type === 'ticket' ? appStore.rightPanelContent.id : null,
  )
  let searchOpen = $state(false)
  const projectHealth = $derived.by(() => {
    const status = data.currentProject?.status?.toLowerCase()
    if (status === 'healthy' || status === 'active') return 'healthy'
    if (status === 'blocked' || status === 'archived') return 'critical'
    return 'degraded'
  })

  const searchCapability = capabilityCatalog.search
  const newTicketCapability = capabilityCatalog.newTicket

  $effect(() => {
    appStore.currentOrg = data.currentOrg
    appStore.currentProject = data.currentProject
  })

  $effect(() => {
    const projectId = data.currentProject?.id
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
    organizations={data.organizations}
    projects={data.projects}
    currentOrgId={data.currentOrg?.id ?? null}
    currentProjectId={data.currentProject?.id ?? null}
    currentSection={data.currentSection}
    orgName={data.currentOrg?.name ?? 'No organization'}
    projectName={data.currentProject?.name ?? ''}
    sseStatus={appStore.sseStatus}
    searchEnabled={searchCapability.state === 'available' && data.organizations.length > 0}
    newTicketEnabled={newTicketCapability.state === 'available' && Boolean(data.currentProject?.id)}
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
        currentOrgId={data.currentOrg?.id ?? null}
        currentProjectId={data.currentProject?.id ?? null}
        projectSelected={Boolean(data.currentProject)}
        projectName={data.currentProject?.name ?? ''}
        {projectHealth}
        agentCount={data.agentCount}
        onToggleCollapse={() => appStore.toggleSidebar()}
      />
    </aside>

    <main class="flex min-w-0 flex-1 flex-col overflow-auto">
      {@render children()}
    </main>
  </div>

  <TicketDrawer
    projectId={data.currentProject?.id}
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
    organizations={data.organizations}
    projects={data.projects}
    currentOrg={data.currentOrg}
    currentProject={data.currentProject}
    currentSection={data.currentSection}
    newTicketEnabled={newTicketCapability.state === 'available' && Boolean(data.currentProject?.id)}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenTicket={(ticketId) => appStore.openRightPanel({ type: 'ticket', id: ticketId })}
  />
</div>
