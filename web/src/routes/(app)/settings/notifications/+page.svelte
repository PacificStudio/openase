<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { createNotificationsController, NotificationsPage } from '$lib/features/notifications'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
  const notifications = createNotificationsController({ workspace })
  let lastSelection = $state('')

  onMount(() => {
    void notifications.start()
  })

  $effect(() => {
    const nextSelection = `${workspace.state.selectedOrgId}:${workspace.state.selectedProjectId}`
    if (!workspace.state.selectedOrgId || nextSelection === lastSelection) {
      return
    }

    lastSelection = nextSelection
    void Promise.all([notifications.refreshChannels(), notifications.refreshRules()])
  })

  onDestroy(() => {
    notifications.destroy()
  })
</script>

<svelte:head>
  <title>Notification Settings · OpenASE</title>
</svelte:head>

<NotificationsPage controller={notifications} />
