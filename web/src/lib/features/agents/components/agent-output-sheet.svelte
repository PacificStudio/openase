<script lang="ts">
  import type { AgentOutputEntry } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
  } from '$ui/sheet'
  import type { AgentInstance } from '../types'

  let {
    open = $bindable(false),
    agent,
    entries,
    loading = false,
    error = '',
    onRefresh,
  }: {
    open?: boolean
    agent: AgentInstance | null
    entries: AgentOutputEntry[]
    loading?: boolean
    error?: string
    onRefresh?: () => void
  } = $props()

  const streamVariants: Record<string, 'default' | 'secondary' | 'outline'> = {
    stdout: 'default',
    stderr: 'secondary',
    system: 'outline',
  }

  function summarizeMetadata(entry: AgentOutputEntry) {
    const pairs = Object.entries(entry.metadata).filter(([key]) => key !== 'stream')
    if (pairs.length === 0) return ''

    return pairs
      .map(([key, value]) => `${key}: ${typeof value === 'string' ? value : JSON.stringify(value)}`)
      .join(' · ')
  }

  function formatTimestamp(value: string) {
    return new Date(value).toLocaleString()
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-2xl">
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <SheetTitle>{agent?.name ?? 'Agent output'}</SheetTitle>
      <SheetDescription>Dedicated runtime output snapshot for the selected agent.</SheetDescription>
    </SheetHeader>

    <div class="border-border flex items-center justify-between border-b px-6 py-3 text-xs">
      <div class="text-muted-foreground flex flex-wrap items-center gap-2">
        <span>Status: {agent?.status ?? 'unknown'}</span>
        <span>Runtime: {agent?.runtimePhase ?? 'unknown'}</span>
        <span>Entries: {entries.length}</span>
      </div>
      <Button variant="outline" size="sm" onclick={onRefresh} disabled={loading || !agent}>
        {loading ? 'Refreshing…' : 'Refresh'}
      </Button>
    </div>

    <div class="flex-1 overflow-y-auto px-6 py-5">
      {#if error}
        <div
          class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
        >
          {error}
        </div>
      {:else if loading && entries.length === 0}
        <div class="text-muted-foreground rounded-md border px-4 py-10 text-center text-sm">
          Loading output…
        </div>
      {:else if entries.length === 0}
        <div class="text-muted-foreground rounded-md border px-4 py-10 text-center text-sm">
          No dedicated output entries have been recorded for this agent yet.
        </div>
      {:else}
        <div class="space-y-3">
          {#each entries as entry (entry.id)}
            <article class="border-border rounded-md border px-4 py-3">
              <div class="flex flex-wrap items-center gap-2 text-xs">
                <Badge variant={streamVariants[entry.stream] ?? 'outline'}>{entry.stream}</Badge>
                <span class="text-muted-foreground">{entry.event_type}</span>
                <span class="text-muted-foreground">{formatTimestamp(entry.created_at)}</span>
              </div>
              <pre
                class="text-foreground mt-3 overflow-x-auto font-mono text-xs break-words whitespace-pre-wrap">{entry.message}</pre>
              {#if summarizeMetadata(entry)}
                <p class="text-muted-foreground mt-3 text-xs">{summarizeMetadata(entry)}</p>
              {/if}
            </article>
          {/each}
        </div>
      {/if}
    </div>

    <SheetFooter class="border-border border-t px-6 py-4">
      <Button variant="outline" onclick={() => (open = false)}>Close</Button>
    </SheetFooter>
  </SheetContent>
</Sheet>
