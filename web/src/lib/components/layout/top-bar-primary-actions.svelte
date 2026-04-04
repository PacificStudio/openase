<script lang="ts">
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { Plus, Search } from '@lucide/svelte'

  let {
    searchEnabled = false,
    newTicketEnabled = false,
    newTicketTitle = '',
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    onOpenSearch,
    onNewTicket,
  }: {
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    newTicketTitle?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    onOpenSearch?: () => void
    onNewTicket?: () => void
  } = $props()
</script>

{#if searchEnabled}
  <Button
    variant="outline"
    size="sm"
    class="text-muted-foreground hidden w-[200px] justify-start gap-2 sm:flex"
    onclick={onOpenSearch}
  >
    <Search class="size-3.5" />
    <span class="text-xs">Search...</span>
    <kbd class="bg-muted ml-auto rounded px-1.5 py-0.5 font-mono text-[10px]">⌘K</kbd>
  </Button>

  <Separator orientation="vertical" class="mx-1 h-5" />
{/if}

<Button
  size="sm"
  class="gap-1.5"
  disabled={!newTicketEnabled}
  title={newTicketEnabled
    ? 'Create ticket'
    : (newTicketTitle ?? 'Ticket creation is not available.')}
  onclick={onNewTicket}
>
  <Plus class="size-3.5" />
  <span class="hidden text-xs sm:inline">New Ticket</span>
</Button>

<div class="text-muted-foreground flex items-center gap-1.5 text-xs" title="SSE: {sseStatus}">
  {#if sseStatus === 'live'}
    <span class="bg-success size-1.5 rounded-full"></span>
  {:else if sseStatus === 'connecting' || sseStatus === 'retrying'}
    <span class="bg-warning size-1.5 animate-pulse rounded-full"></span>
  {:else}
    <span class="bg-destructive size-1.5 rounded-full"></span>
  {/if}
</div>
