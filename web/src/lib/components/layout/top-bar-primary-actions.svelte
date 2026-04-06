<script lang="ts">
  import { viewport } from '$lib/stores/viewport.svelte'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { Bot, Plus, Search } from '@lucide/svelte'

  let {
    searchEnabled = false,
    newTicketEnabled = false,
    newTicketTitle = '',
    sseStatus = 'live' as 'idle' | 'connecting' | 'live' | 'retrying',
    projectSelected = false,
    onOpenSearch,
    onNewTicket,
    onOpenProjectAssistant,
  }: {
    searchEnabled?: boolean
    newTicketEnabled?: boolean
    newTicketTitle?: string
    sseStatus?: 'idle' | 'connecting' | 'live' | 'retrying'
    projectSelected?: boolean
    onOpenSearch?: () => void
    onNewTicket?: () => void
    onOpenProjectAssistant?: (initialPrompt?: string) => void
  } = $props()

  const isMobile = $derived(viewport.isMobile)
</script>

{#if searchEnabled}
  {#if isMobile}
    <Button variant="ghost" size="icon-sm" onclick={onOpenSearch} aria-label="Search">
      <Search class="size-4" />
    </Button>
  {:else}
    <Button
      variant="outline"
      size="sm"
      class="text-muted-foreground w-[200px] justify-start gap-2"
      onclick={onOpenSearch}
    >
      <Search class="size-3.5" />
      <span class="text-xs">Search...</span>
      <kbd class="bg-muted ml-auto rounded px-1.5 py-0.5 font-mono text-[10px]">⌘K</kbd>
    </Button>
    <Separator orientation="vertical" class="mx-1 h-5" />
  {/if}
{/if}

{#if isMobile && projectSelected}
  <Button
    variant="ghost"
    size="icon-sm"
    onclick={() => onOpenProjectAssistant?.()}
    aria-label="Project AI"
  >
    <Bot class="size-4" />
  </Button>
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
