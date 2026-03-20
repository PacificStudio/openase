<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { ConnectorsPage, createConnectorsController } from '$lib/features/connectors'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
  const connectors = createConnectorsController({ workspace })
  let lastProjectId = $state('')

  onMount(() => {
    void connectors.start()
  })

  $effect(() => {
    const projectId = workspace.state.selectedProjectId
    if (!projectId || projectId === lastProjectId) {
      return
    }

    lastProjectId = projectId
    void connectors.refreshConnectors()
  })

  onDestroy(() => {
    connectors.destroy()
  })
</script>

<svelte:head>
  <title>Connector Settings · OpenASE</title>
</svelte:head>

<ConnectorsPage controller={connectors} />
