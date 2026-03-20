<script lang="ts">
  import { page } from '$app/state'
  import { onDestroy } from 'svelte'
  import { TicketDetailPage, createTicketDetailStore } from '$lib/features/ticket-detail'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
  const ticketDetail = createTicketDetailStore()
  let lastSearch = $state('')

  $effect(() => {
    const search = page.url.search
    if (search === lastSearch) {
      return
    }

    lastSearch = search
    void ticketDetail.start({
      projectId:
        page.url.searchParams.get('project')?.trim() ?? workspace.state.selectedProjectId ?? '',
      ticketId: page.url.searchParams.get('id')?.trim() ?? '',
    })
  })

  onDestroy(() => {
    ticketDetail.destroy()
  })
</script>

<TicketDetailPage controller={ticketDetail} />
