<script lang="ts">
  import { getWorkspaceContext } from '$lib/features/workspace'
  import { WorkflowsPage } from '$lib/features/workflows'

  const workspace = getWorkspaceContext()
</script>

<svelte:head>
  <title>Workflows · OpenASE</title>
</svelte:head>

<WorkflowsPage controller={workspace} />
