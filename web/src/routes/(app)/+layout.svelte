<script lang="ts">
  import { cn } from '$lib/utils'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import { page } from '$app/state'

  let { children } = $props()

  let sidebarCollapsed = $state(false)

  let currentPath = $derived(page.url.pathname)

  function handleOpenSearch() {
    // TODO: open command palette
  }

  function handleNewTicket() {
    // TODO: open new ticket dialog
  }

  function handleToggleTheme() {
    document.documentElement.classList.toggle('dark')
  }
</script>

<div class="flex h-screen flex-col overflow-hidden bg-background">
  <TopBar
    orgName="My Org"
    projectName="Alpha Platform"
    sseStatus="live"
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
  />

  <div class="flex flex-1 overflow-hidden">
    <!-- Sidebar -->
    <aside
      class={cn(
        'flex h-full flex-col border-r border-border bg-sidebar transition-[width] duration-200 ease-in-out',
        sidebarCollapsed ? 'w-[52px]' : 'w-[240px]',
      )}
    >
      <Sidebar
        collapsed={sidebarCollapsed}
        currentPath={currentPath}
        projectSelected={true}
        projectName="Alpha Platform"
        projectHealth="healthy"
        approvalCount={3}
        agentCount={4}
        onToggleCollapse={() => (sidebarCollapsed = !sidebarCollapsed)}
      />
    </aside>

    <!-- Main Content -->
    <main class="flex min-w-0 flex-1 flex-col overflow-auto">
      {@render children()}
    </main>
  </div>
</div>
