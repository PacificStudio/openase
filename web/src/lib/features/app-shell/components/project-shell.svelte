<script lang="ts">
  import { goto } from '$app/navigation'
  import { page } from '$app/state'
  import type { Organization, Project } from '$lib/api/contracts'
  import { connectEventStream } from '$lib/api/sse'
  import Sidebar from '$lib/components/layout/sidebar.svelte'
  import TopBar from '$lib/components/layout/top-bar.svelte'
  import { capabilityCatalog } from '$lib/features/capabilities'
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
  }

  let { children, data }: { children: Snippet; data: ShellData } = $props()

  let currentPath = $derived(page.url.pathname)
  let currentTicketId = $derived(
    appStore.rightPanelContent?.type === 'ticket' ? appStore.rightPanelContent.id : null,
  )
  let organizationOptions = $derived(
    data.organizations.map((organization) => ({
      id: organization.id,
      name: organization.name,
    })),
  )
  let projectOptions = $derived(
    data.projects.map((project) => ({
      id: project.id,
      name: project.name,
    })),
  )

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

  function handleOpenSearch() {
    return
  }

  function handleNewTicket() {
    appStore.openNewTicketDialog()
  }

  function handleToggleTheme() {
    appStore.toggleTheme()
  }

  function buildContextHref(orgId: string | null, projectId: string | null) {
    const nextUrl = new URL(page.url)

    if (orgId) {
      nextUrl.searchParams.set('orgId', orgId)
    } else {
      nextUrl.searchParams.delete('orgId')
    }

    if (projectId) {
      nextUrl.searchParams.set('projectId', projectId)
    } else {
      nextUrl.searchParams.delete('projectId')
    }

    return `${nextUrl.pathname}${nextUrl.search}${nextUrl.hash}`
  }

  function navigateToContext(orgId: string | null, projectId: string | null, replaceState = false) {
    const nextHref = buildContextHref(orgId, projectId)
    const currentHref = `${page.url.pathname}${page.url.search}${page.url.hash}`

    if (nextHref === currentHref) {
      return
    }

    void goto(nextHref, {
      replaceState,
      noScroll: true,
      keepFocus: true,
    })
  }

  function handleSelectOrg(orgId: string) {
    if (orgId === data.currentOrg?.id) {
      return
    }

    navigateToContext(orgId, null)
  }

  function handleSelectProject(projectId: string) {
    if (projectId === data.currentProject?.id) {
      return
    }

    navigateToContext(data.currentOrg?.id ?? null, projectId)
  }
</script>

<div class="bg-background flex h-screen flex-col overflow-hidden">
  <TopBar
    orgName={data.currentOrg?.name ?? 'No organization'}
    projectName={data.currentProject?.name ?? ''}
    organizations={organizationOptions}
    projects={projectOptions}
    currentOrgId={data.currentOrg?.id ?? null}
    currentProjectId={data.currentProject?.id ?? null}
    sseStatus={appStore.sseStatus}
    searchEnabled={searchCapability.state === 'available'}
    newTicketEnabled={newTicketCapability.state === 'available' && Boolean(data.currentProject?.id)}
    searchTitle={searchCapability.summary}
    newTicketTitle={newTicketCapability.summary}
    onToggleTheme={handleToggleTheme}
    onNewTicket={handleNewTicket}
    onOpenSearch={handleOpenSearch}
    onSelectOrg={handleSelectOrg}
    onSelectProject={handleSelectProject}
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
</div>
