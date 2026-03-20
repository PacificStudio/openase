<script lang="ts">
  import { page } from '$app/state'
  import { onDestroy } from 'svelte'
  import AppShell from '$lib/components/layout/AppShell.svelte'
  import RightDrawer from '$lib/components/layout/RightDrawer.svelte'
  import Sidebar from '$lib/components/layout/Sidebar.svelte'
  import TopBar from '$lib/components/layout/TopBar.svelte'
  import { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import { setWorkspaceContext } from '$lib/features/workspace/context'
  import { readWorkspaceRouteSelection } from '$lib/features/workspace/routing'
  import WorkspaceContextDrawer from '$lib/features/workspace/components/WorkspaceContextDrawer.svelte'

  let { children } = $props()

  const workspace = setWorkspaceContext(createWorkspaceController())
  let sidebarOpen = $state(false)
  let lastSearch = $state('')

  const shellState = $derived.by(() => describeShell(page.url.pathname))
  const shellStatus = $derived.by(() => {
    if (workspace.state.errorMessage) {
      return {
        pageStatus: 'error',
        statusMessage: 'API issue',
      }
    }

    const states = [
      workspace.board.streamState,
      workspace.dashboard.agentStreamState,
      workspace.dashboard.activityStreamState,
    ]

    if (states.includes('retrying')) {
      return {
        pageStatus: 'degraded',
        statusMessage: 'Stream warning',
      }
    }

    if (states.includes('connecting')) {
      return {
        pageStatus: 'reconnecting',
        statusMessage: 'Reconnecting',
      }
    }

    if (states.includes('live')) {
      return {
        pageStatus: 'live',
        statusMessage: 'SSE live',
      }
    }

    return {
      pageStatus: 'idle',
      statusMessage: '',
    }
  })

  const runningAgentCount = $derived(
    workspace.dashboard.agents.filter((agent) => agent.status === 'running').length,
  )

  $effect(() => {
    const search = page.url.search
    if (search === lastSearch) {
      return
    }

    lastSearch = search
    sidebarOpen = false
    void workspace.start(readWorkspaceRouteSelection(page.url.searchParams))
  })

  onDestroy(() => {
    workspace.destroy()
  })

  function closeSidebar() {
    sidebarOpen = false
  }

  function toggleSidebar() {
    sidebarOpen = !sidebarOpen
  }

  function closeDrawer() {
    workspace.toggleDrawer(false)
  }

  function describeShell(pathname: string) {
    if (pathname.startsWith('/board')) {
      return {
        selectedPage: 'board' as const,
        pageTitle: 'Board',
        pageLabel: 'Default work view',
        drawerTitle: 'Project context',
        drawerDescription: 'Keep org and project setup close while the board stays focused on ticket flow.',
      }
    }

    if (pathname.startsWith('/workflows')) {
      return {
        selectedPage: 'workflows' as const,
        pageTitle: 'Workflows',
        pageLabel: 'Harness control',
        drawerTitle: 'Project context',
        drawerDescription: 'Tune project context and setup without leaving the workflow editor.',
      }
    }

    if (pathname.startsWith('/agents')) {
      return {
        selectedPage: 'agents' as const,
        pageTitle: 'Agents',
        pageLabel: 'Runtime operations',
        drawerTitle: 'Project context',
        drawerDescription: 'Keep project scope visible while drilling into live agent health.',
      }
    }

    if (pathname.startsWith('/activity')) {
      return {
        selectedPage: 'activity' as const,
        pageTitle: 'Activity',
        pageLabel: 'Audit stream',
        drawerTitle: 'Project context',
        drawerDescription: 'Keep the current org/project scope editable while reviewing execution history.',
      }
    }

    if (pathname.startsWith('/settings')) {
      return {
        selectedPage: 'settings' as const,
        pageTitle: 'Settings',
        pageLabel: 'Configuration surfaces',
        drawerTitle: 'Project context',
        drawerDescription: 'Separate runtime observation from project configuration and repo wiring.',
      }
    }

    if (pathname.startsWith('/ticket')) {
      return {
        selectedPage: 'board' as const,
        pageTitle: 'Ticket detail',
        pageLabel: 'Drawer-first deep dive',
        drawerTitle: 'Project context',
        drawerDescription: 'Stay anchored to the same project while inspecting a ticket in depth.',
      }
    }

    return {
      selectedPage: 'overview' as const,
      pageTitle: 'Overview',
      pageLabel: 'Project dashboard',
      drawerTitle: 'Project context',
      drawerDescription: 'Keep org and project setup editable without leaving the default dashboard.',
    }
  }
</script>

{#snippet header()}
  <TopBar
    selectedOrg={workspace.state.selectedOrg}
    selectedProject={workspace.state.selectedProject}
    pageTitle={shellState.pageTitle}
    pageLabel={shellState.pageLabel}
    pageStatus={shellStatus.pageStatus}
    runningAgentCount={runningAgentCount}
    ticketCount={workspace.board.tickets.length}
    workflowCount={workspace.state.workflows.length}
    statusMessage={shellStatus.statusMessage}
    onToggleSidebar={toggleSidebar}
    onToggleDrawer={() => workspace.toggleDrawer()}
  />
{/snippet}

{#snippet sidebar()}
  <Sidebar
    organizations={workspace.state.organizations}
    projects={workspace.state.projects}
    selectedOrgId={workspace.state.selectedOrgId}
    selectedProjectId={workspace.state.selectedProjectId}
    selectedPage={shellState.selectedPage}
    workflowCount={workspace.state.workflows.length}
    ticketCount={workspace.board.tickets.length}
    runningAgentCount={runningAgentCount}
    connectorCount={0}
    onSelectOrganization={workspace.selectOrganization}
    onSelectProject={workspace.selectProject}
  />
{/snippet}

{#snippet drawer()}
  <RightDrawer title={shellState.drawerTitle} description={shellState.drawerDescription}>
    <WorkspaceContextDrawer controller={workspace} />
  </RightDrawer>
{/snippet}

<AppShell
  {header}
  {sidebar}
  {drawer}
  {sidebarOpen}
  drawerOpen={workspace.state.drawerOpen}
  onCloseSidebar={closeSidebar}
  onCloseDrawer={closeDrawer}
>
  {@render children()}
</AppShell>
