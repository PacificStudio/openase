<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { AgentsPage, createAgentsController } from '$lib/features/agents'
  import {
    readWorkspaceRouteSelection,
    WorkspaceContextDrawer,
    WorkspacePageShell,
  } from '$lib/features/workspace'

  const agents = createAgentsController()

  onMount(() => {
    void agents.start(readWorkspaceRouteSelection(new URLSearchParams(window.location.search)))
  })

  onDestroy(() => {
    agents.destroy()
  })
</script>

{#snippet drawerPane()}
  <WorkspaceContextDrawer controller={agents.workspace} />
{/snippet}

<WorkspacePageShell
  workspace={agents.workspace}
  selectedPage="agents"
  drawerTitle="Project context"
  drawerDescription="Keep organization and project scope editable while provider and agent telemetry stays on the agents page."
  {drawerPane}
>
  <AgentsPage controller={agents} />
</WorkspacePageShell>
