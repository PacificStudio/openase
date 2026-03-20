<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { WorkflowsPage } from '$lib/features/workflows'
  import {
    createWorkspaceController,
    readWorkspaceRouteSelection,
    WorkspaceContextDrawer,
    WorkspacePageShell,
  } from '$lib/features/workspace'

  const workspace = createWorkspaceController()

  onMount(() => {
    void workspace.start(readWorkspaceRouteSelection(new URLSearchParams(window.location.search)))
  })

  onDestroy(() => {
    workspace.destroy()
  })
</script>

{#snippet drawerPane()}
  <WorkspaceContextDrawer controller={workspace} />
{/snippet}

<WorkspacePageShell
  {workspace}
  selectedPage="workflows"
  drawerTitle="Project context"
  drawerDescription="Keep organization and project scope editable while workflow logic lives in the workflows feature."
  {drawerPane}
>
  <WorkflowsPage controller={workspace} />
</WorkspacePageShell>
