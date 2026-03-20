<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { AgentsPage, createAgentsController } from '$lib/features/agents'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
  const agents = createAgentsController({ workspace })
  let lastOrgId = $state('')

  onMount(() => {
    void agents.start()
  })

  $effect(() => {
    const orgId = workspace.state.selectedOrgId
    if (!orgId || orgId === lastOrgId) {
      return
    }

    lastOrgId = orgId
    void agents.refreshProviders()
  })

  onDestroy(() => {
    agents.destroy()
  })
</script>

<AgentsPage controller={agents} />
