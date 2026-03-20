<script lang="ts">
  import type { Snippet } from 'svelte'
  import AppShell from '$lib/components/layout/AppShell.svelte'
  import RightDrawer from '$lib/components/layout/RightDrawer.svelte'
  import Sidebar from '$lib/components/layout/Sidebar.svelte'
  import TopBar from '$lib/components/layout/TopBar.svelte'
  import type { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import WorkspaceControlDrawer from './WorkspaceControlDrawer.svelte'

  let {
    workspace,
    selectedPage = 'board',
    drawerTitle = 'Workspace drawer',
    drawerDescription = 'Keep project setup, workflow tuning, and harness editing off the route layer.',
    drawerPane,
    children,
  }: {
    workspace: ReturnType<typeof createWorkspaceController>
    selectedPage?: 'board' | 'workflows' | 'agents' | 'notifications'
    drawerTitle?: string
    drawerDescription?: string
    drawerPane?: Snippet
    children?: Snippet
  } = $props()
</script>

{#snippet header()}
  <TopBar
    selectedOrg={workspace.state.selectedOrg}
    selectedProject={workspace.state.selectedProject}
    notice={workspace.state.notice}
    errorMessage={workspace.state.errorMessage}
    onToggleDrawer={() => workspace.toggleDrawer()}
  />
{/snippet}

{#snippet sidebar()}
  <Sidebar
    organizations={workspace.state.organizations}
    projects={workspace.state.projects}
    selectedOrgId={workspace.state.selectedOrgId}
    selectedProjectId={workspace.state.selectedProjectId}
    workflowCount={workspace.state.workflows.length}
    ticketCount={workspace.board.tickets.length}
    {selectedPage}
    onSelectOrganization={workspace.selectOrganization}
    onSelectProject={workspace.selectProject}
  />
{/snippet}

{#snippet drawer()}
  <RightDrawer title={drawerTitle} description={drawerDescription}>
    {#if drawerPane}
      {@render drawerPane()}
    {:else}
      <WorkspaceControlDrawer controller={workspace} />
    {/if}
  </RightDrawer>
{/snippet}

<AppShell {header} {sidebar} {drawer} drawerOpen={workspace.state.drawerOpen}>
  {@render children?.()}
</AppShell>
