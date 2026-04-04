<script lang="ts">
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Search, Columns3, List, AlertTriangle, EyeOff } from '@lucide/svelte'
  import { parseBoardFilterPriority } from '../priority'
  import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
  import PriorityIcon from './priority-icon.svelte'
  import type { BoardFilter } from '../types'

  let {
    filter = { search: '' },
    hideEmpty = false,
    workflows = [],
    agents = [],
    class: className = '',
    onFilterChange,
    onHideEmptyChange,
  }: {
    filter?: BoardFilter
    hideEmpty?: boolean
    workflows?: string[]
    agents?: string[]
    class?: string
    onFilterChange?: (next: BoardFilter) => void
    onHideEmptyChange?: (next: boolean) => void
  } = $props()

  function updateFilter(next: BoardFilter) {
    onFilterChange?.(next)
  }

  const parsePriorityFilter = parseBoardFilterPriority
</script>

<div class={cn('flex items-center gap-2', className)}>
  <div class="relative w-52">
    <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
    <Input
      type="text"
      placeholder="Search tickets..."
      class="h-8 pl-8 text-sm"
      value={filter.search ?? ''}
      oninput={(event) =>
        updateFilter({ ...filter, search: (event.currentTarget as HTMLInputElement).value })}
    />
  </div>

  <Select.Root
    type="single"
    onValueChange={(v) => {
      updateFilter({ ...filter, workflow: v || undefined })
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
        updateFilter({ ...filter, agent: v || undefined })
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
      updateFilter({ ...filter, priority: parsePriorityFilter(v) })
    }}
  >
    <Select.Trigger size="sm" class="h-8 text-xs">
      {#if filter.priority}
        <span class="flex items-center gap-1.5">
          <PriorityIcon priority={filter.priority} />
          <span class="capitalize">{filter.priority}</span>
        </span>
      {:else}
        Priority
      {/if}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="">All</Select.Item>
      <Select.Item value="urgent"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="urgent" /><span>Urgent</span></span
        ></Select.Item
      >
      <Select.Item value="high"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="high" /><span>High</span></span
        ></Select.Item
      >
      <Select.Item value="medium"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="medium" /><span>Medium</span></span
        ></Select.Item
      >
      <Select.Item value="low"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="low" /><span>Low</span></span
        ></Select.Item
      >
    </Select.Content>
  </Select.Root>

  <Button
    variant={filter.anomalyOnly ? 'secondary' : 'ghost'}
    size="sm"
    class="h-8 gap-1 text-xs"
    onclick={() => {
      updateFilter({ ...filter, anomalyOnly: !filter.anomalyOnly })
    }}
  >
    <AlertTriangle class="size-3" />
    Anomalies
  </Button>

  {#if ticketViewStore.mode === 'board'}
    <Button
      variant={hideEmpty ? 'secondary' : 'ghost'}
      size="sm"
      class="h-8 gap-1 text-xs"
      onclick={() => {
        onHideEmptyChange?.(!hideEmpty)
      }}
    >
      <EyeOff class="size-3" />
      Hide empty
    </Button>
  {/if}

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
