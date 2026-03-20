<script lang="ts">
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Search, Columns3, List, AlertTriangle } from '@lucide/svelte'
  import type { BoardFilter } from '../types'

  let {
    filter = $bindable({ search: '' }),
    view = $bindable<'board' | 'list'>('board'),
    workflows = [],
    agents = [],
    class: className = '',
  }: {
    filter?: BoardFilter
    view?: 'board' | 'list'
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
    <Search class="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
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
    onValueChange={(v) => { filter = { ...filter, workflow: v || undefined } }}
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

  <Select.Root
    type="single"
    onValueChange={(v) => { filter = { ...filter, agent: v || undefined } }}
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

  <Select.Root
    type="single"
    onValueChange={(v) => { filter = { ...filter, priority: v || undefined } }}
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
    onclick={() => { filter = { ...filter, anomalyOnly: !filter.anomalyOnly } }}
  >
    <AlertTriangle class="size-3" />
    Anomalies
  </Button>

  <div class="ml-auto flex items-center rounded-md border border-border">
    <Button
      variant={view === 'board' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-r-none px-2"
      onclick={() => { view = 'board' }}
    >
      <Columns3 class="size-3.5" />
    </Button>
    <Button
      variant={view === 'list' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-l-none px-2"
      onclick={() => { view = 'list' }}
    >
      <List class="size-3.5" />
    </Button>
  </div>
</div>
