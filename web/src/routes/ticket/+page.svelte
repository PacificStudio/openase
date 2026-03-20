<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { createTicketDetailStore, TicketDetailPage } from '$lib/features/ticket-detail'

  const ticketDetail = createTicketDetailStore()

  onMount(() => {
    const searchParams = new URLSearchParams(window.location.search)
    void ticketDetail.start({
      projectId: searchParams.get('project') ?? '',
      ticketId: searchParams.get('id') ?? '',
    })
  })

  onDestroy(() => {
    ticketDetail.destroy()
  })
</script>

<TicketDetailPage controller={ticketDetail} />
