<script lang="ts">
  import type { Snippet } from 'svelte'
  import AppShell from '$lib/components/layout/AppShell.svelte'
  import RightDrawer from '$lib/components/layout/RightDrawer.svelte'
  import Sidebar from '$lib/components/layout/Sidebar.svelte'
  import TopBar from '$lib/components/layout/TopBar.svelte'
  import type { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import type { Organization, Project } from '$lib/features/workspace/types'
  import WorkspaceControlDrawer from './WorkspaceControlDrawer.svelte'

  let {
    workspace,
    selectedPage = 'board',
    drawerTitle = 'Workspace drawer',
    drawerDescription = 'Keep project setup, workflow tuning, and harness editing off the route layer.',
    headerTags,
    headerEyebrow,
    headerTitle,
    headerDescription,
    connectorCount = 0,
    onSelectOrganization,
    onSelectProject,
    drawerPane,
    children,
  }: {
    workspace: ReturnType<typeof createWorkspaceController>
    selectedPage?: 'board' | 'workflows' | 'agents' | 'connectors'
    drawerTitle?: string
    drawerDescription?: string
    headerTags?: string[]
    headerEyebrow?: string
    headerTitle?: string
    headerDescription?: string
    connectorCount?: number
    onSelectOrganization?: (organization: Organization) => void | Promise<void>
    onSelectProject?: (project: Project) => void | Promise<void>
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
    pageTags={headerTags}
    pageEyebrow={headerEyebrow}
    pageTitle={headerTitle}
    pageDescription={headerDescription}
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
    {connectorCount}
    {selectedPage}
    onSelectOrganization={onSelectOrganization ?? workspace.selectOrganization}
    onSelectProject={onSelectProject ?? workspace.selectProject}
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
