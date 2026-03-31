<script lang="ts">
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Search, Columns3, List, AlertTriangle } from '@lucide/svelte'
  import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
  import type { BoardFilter } from '../types'

  let {
    filter = $bindable({ search: '' }),
    workflows = [],
    agents = [],
    class: className = '',
  }: {
    filter?: BoardFilter
    workflows?: string[]
    agents?: string[]
    class?: string
  } = $props()

  let searchValue = $state(filter.search ?? '')

  function handleSearch() {
    filter = { ...filter, search: searchValue }
  }
</script>

<div class={cn('flex items-center gap-2', className)}>
  <div class="relative w-52">
    <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
    <Input
      type="text"
      placeholder="Search tickets..."
      class="h-8 pl-8 text-sm"
      bind:value={searchValue}
      oninput={handleSearch}
    />
  </div>

  <Select.Root
    type="single"
    onValueChange={(v) => {
      filter = { ...filter, workflow: v || undefined }
    }}
  >
    <Select.Trigger size="sm" class="h-8 text-xs">
      {filter.workflow ?? 'Workflow'}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="">All</Select.Item>
      {#each workflows as wf}
        <Select.Item value={wf}>{wf}</Select.Item>
      {/each}
    </Select.Content>
  </Select.Root>

  {#if agents.length > 0}
    <Select.Root
      type="single"
      onValueChange={(v) => {
        filter = { ...filter, agent: v || undefined }
      }}
    >
      <Select.Trigger size="sm" class="h-8 text-xs">
        {filter.agent ?? 'Agent'}
      </Select.Trigger>
      <Select.Content>
        <Select.Item value="">All</Select.Item>
        {#each agents as a}
          <Select.Item value={a}>{a}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  {/if}

  <Select.Root
    type="single"
    onValueChange={(v) => {
      filter = { ...filter, priority: v || undefined }
    }}
  >
    <Select.Trigger size="sm" class="h-8 text-xs">
      {filter.priority ?? 'Priority'}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="">All</Select.Item>
      <Select.Item value="urgent">Urgent</Select.Item>
      <Select.Item value="high">High</Select.Item>
      <Select.Item value="medium">Medium</Select.Item>
      <Select.Item value="low">Low</Select.Item>
    </Select.Content>
  </Select.Root>

  <Button
    variant={filter.anomalyOnly ? 'secondary' : 'ghost'}
    size="sm"
    class="h-8 gap-1 text-xs"
    onclick={() => {
      filter = { ...filter, anomalyOnly: !filter.anomalyOnly }
    }}
  >
    <AlertTriangle class="size-3" />
    Anomalies
  </Button>

  <div class="border-border ml-auto flex items-center rounded-md border">
    <Button
      variant={ticketViewStore.mode === 'board' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-r-none px-2"
      aria-label="Board view"
      onclick={() => ticketViewStore.setMode('board')}
    >
      <Columns3 class="size-3.5" />
    </Button>
    <Button
      variant={ticketViewStore.mode === 'list' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-l-none px-2"
      aria-label="List view"
      onclick={() => ticketViewStore.setMode('list')}
    >
      <List class="size-3.5" />
    </Button>
  </div>
</div>
