<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { ConnectorsPage, createConnectorsController } from '$lib/features/connectors'
  import {
    readWorkspaceRouteSelection,
    WorkspaceContextDrawer,
    WorkspacePageShell,
  } from '$lib/features/workspace'

  const connectors = createConnectorsController()

  onMount(() => {
    void connectors.start(readWorkspaceRouteSelection(new URLSearchParams(window.location.search)))
  })

  onDestroy(() => {
    connectors.destroy()
  })
</script>

{#snippet drawerPane()}
  <WorkspaceContextDrawer controller={connectors.workspace} />
{/snippet}

<WorkspacePageShell
  workspace={connectors.workspace}
  selectedPage="connectors"
  connectorCount={connectors.connectors.length}
  headerTags={['Phase 2 UI', 'Settings', 'Connectors', connectors.persistenceMode === 'api' ? 'API-backed' : 'Local draft']}
  headerEyebrow="OpenASE connector control"
  headerTitle="Manage issue connectors without leaving the project shell."
  headerDescription="Create connector configs, inspect sync health, and trigger test or sync actions from a dedicated settings surface that matches the Phase 2 frontend split."
  drawerTitle="Project context"
  drawerDescription="Keep organization and project scope editable while connector operations stay in their own feature slice."
  onSelectOrganization={connectors.selectOrganization}
  onSelectProject={connectors.selectProject}
  {drawerPane}
>
  <ConnectorsPage controller={connectors} />
</WorkspacePageShell>
