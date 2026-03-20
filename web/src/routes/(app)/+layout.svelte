<script lang="ts">
  import { cn } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import TicketDrawer from '$lib/features/ticket-detail/components/ticket-drawer.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import { page } from '$app/state'
  import type { Snippet } from 'svelte'
  import type { LayoutData } from './$types'

  let { children, data }: { children: Snippet; data: LayoutData } = $props()

  let currentPath = $derived(page.url.pathname)
  let currentTicketId = $derived(
    appStore.rightPanelContent?.type === 'ticket' ? appStore.rightPanelContent.id : null,
  )

  const projectHealth = $derived.by(() => {
    const status = data.currentProject?.status?.toLowerCase()
    if (status === 'healthy' || status === 'active') return 'healthy'
    if (status === 'blocked' || status === 'archived') return 'critical'
    return 'degraded'
  })

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

  function handleOpenSearch() {
    return
  }

  function handleNewTicket() {
    return
  }

  function handleToggleTheme() {
    appStore.toggleTheme()
  }
</script>

<div class="flex h-screen flex-col overflow-hidden bg-background">
  <TopBar
    orgName={data.currentOrg?.name ?? 'No organization'}
    projectName={data.currentProject?.name ?? ''}
    sseStatus={appStore.sseStatus}
    searchEnabled={false}
    newTicketEnabled={false}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
  />

  <div class="flex flex-1 overflow-hidden">
    <!-- Sidebar -->
    <aside
      class={cn(
        'flex h-full flex-col border-r border-border bg-sidebar transition-[width] duration-200 ease-in-out',
        appStore.sidebarCollapsed ? 'w-[52px]' : 'w-[240px]',
      )}
    >
      <Sidebar
        collapsed={appStore.sidebarCollapsed}
        currentPath={currentPath}
        projectSelected={Boolean(data.currentProject)}
        projectName={data.currentProject?.name ?? ''}
        projectHealth={projectHealth}
        agentCount={data.agentCount}
        onToggleCollapse={() => appStore.toggleSidebar()}
      />
    </aside>

    <!-- Main Content -->
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
</div>
