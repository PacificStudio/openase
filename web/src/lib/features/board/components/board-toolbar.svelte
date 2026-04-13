<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { cn } from '$lib/utils'
  import { Input } from '$ui/input'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Search, Columns3, List, AlertTriangle, EyeOff } from '@lucide/svelte'
  import { parseBoardFilterPriority } from '../priority'
  import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
  import PriorityIcon from './priority-icon.svelte'
  import type { BoardFilter } from '../types'
  import { formatBoardPriorityLabel } from '../priority'

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

<div class={cn('flex flex-wrap items-center gap-2', className)} data-tour="board-toolbar">
  <div class="relative min-w-0 flex-1 basis-full sm:flex-none sm:basis-52">
    <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2" />
    <Input
      type="text"
      placeholder={i18nStore.t('board.searchTicketsPlaceholder')}
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
    <Select.Trigger size="sm" class="h-8 min-w-[7rem] text-xs">
      {filter.workflow ?? i18nStore.t('common.workflow')}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="">{i18nStore.t('common.all')}</Select.Item>
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
      <Select.Trigger size="sm" class="h-8 min-w-[7rem] text-xs">
        {filter.agent ?? i18nStore.t('common.agent')}
      </Select.Trigger>
      <Select.Content>
        <Select.Item value="">{i18nStore.t('common.all')}</Select.Item>
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
    <Select.Trigger size="sm" class="h-8 min-w-[7rem] text-xs">
      {#if filter.priority}
        <span class="flex items-center gap-1.5">
          <PriorityIcon priority={filter.priority} />
          <span>{formatBoardPriorityLabel(filter.priority, i18nStore.locale)}</span>
        </span>
      {:else}
        {i18nStore.t('common.priority')}
      {/if}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="">{i18nStore.t('common.all')}</Select.Item>
      <Select.Item value="urgent"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="urgent" /><span
            >{formatBoardPriorityLabel('urgent', i18nStore.locale)}</span
          ></span
        ></Select.Item
      >
      <Select.Item value="high"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="high" /><span
            >{formatBoardPriorityLabel('high', i18nStore.locale)}</span
          ></span
        ></Select.Item
      >
      <Select.Item value="medium"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="medium" /><span
            >{formatBoardPriorityLabel('medium', i18nStore.locale)}</span
          ></span
        ></Select.Item
      >
      <Select.Item value="low"
        ><span class="flex items-center gap-1.5"
          ><PriorityIcon priority="low" /><span
            >{formatBoardPriorityLabel('low', i18nStore.locale)}</span
          ></span
        ></Select.Item
      >
    </Select.Content>
  </Select.Root>

  <Button
    variant={filter.anomalyOnly ? 'secondary' : 'ghost'}
    size="sm"
    class="h-8 shrink-0 gap-1 text-xs"
    onclick={() => {
      updateFilter({ ...filter, anomalyOnly: !filter.anomalyOnly })
    }}
  >
    <AlertTriangle class="size-3" />
    {i18nStore.t('board.anomalies')}
  </Button>

  {#if ticketViewStore.mode === 'board'}
    <Button
      variant={hideEmpty ? 'secondary' : 'ghost'}
      size="sm"
      class="h-8 shrink-0 gap-1 text-xs"
      onclick={() => {
        onHideEmptyChange?.(!hideEmpty)
      }}
    >
      <EyeOff class="size-3" />
      {i18nStore.t('board.hideEmpty')}
    </Button>
  {/if}

  <div class="border-border ml-auto flex shrink-0 items-center rounded-md border" data-tour="board-view-toggle">
    <Button
      variant={ticketViewStore.mode === 'board' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-r-none px-2"
      aria-label={i18nStore.t('board.boardView')}
      onclick={() => ticketViewStore.setMode('board')}
    >
      <Columns3 class="size-3.5" />
    </Button>
    <Button
      variant={ticketViewStore.mode === 'list' ? 'secondary' : 'ghost'}
      size="sm"
      class="h-7 rounded-l-none px-2"
      aria-label={i18nStore.t('board.listView')}
      onclick={() => ticketViewStore.setMode('list')}
    >
      <List class="size-3.5" />
    </Button>
  </div>
</div>
