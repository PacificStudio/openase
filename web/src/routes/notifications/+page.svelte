<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { createNotificationsController, NotificationsPage } from '$lib/features/notifications'
  import {
    readWorkspaceRouteSelection,
    WorkspaceContextDrawer,
    WorkspacePageShell,
  } from '$lib/features/workspace'

  const notifications = createNotificationsController()

  onMount(() => {
    void notifications.start(
      readWorkspaceRouteSelection(new URLSearchParams(window.location.search)),
    )
  })

  onDestroy(() => {
    notifications.destroy()
  })
</script>

{#snippet drawerPane()}
  <WorkspaceContextDrawer controller={notifications.workspace} />
{/snippet}

<WorkspacePageShell
  workspace={notifications.workspace}
  selectedPage="notifications"
  drawerTitle="Project context"
  drawerDescription="Keep organization and project scope editable while notification channel and rule management stays inside the notifications feature."
  {drawerPane}
>
  <NotificationsPage controller={notifications} />
</WorkspacePageShell>
