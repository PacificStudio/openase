<script lang="ts">
  import { TicketsPage } from '$lib/features/tickets'
  import { TicketDrawer } from '$lib/features/ticket-detail'
  import { appStore } from '$lib/stores/app.svelte'

  const currentTicketId = $derived(
    appStore.rightPanelContent?.type === 'ticket' ? appStore.rightPanelContent.id : null,
  )
</script>

<div class="flex h-full min-h-0 flex-col">
  <TicketsPage />
  <TicketDrawer
    projectId={appStore.currentProject?.id}
    ticketId={currentTicketId}
    open={appStore.rightPanelOpen}
    onOpenChange={(open) => {
      if (!open) {
        appStore.closeRightPanel()
      }
    }}
  />
</div>
