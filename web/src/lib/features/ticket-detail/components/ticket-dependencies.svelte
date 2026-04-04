<script lang="ts">
  import { Badge } from '$ui/badge'
  import { buttonVariants, Button } from '$ui/button'
  import * as Popover from '$ui/popover'
  import * as Command from '$ui/command'
  import Plus from '@lucide/svelte/icons/plus'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import ArrowLeft from '@lucide/svelte/icons/arrow-left'
  import CircleDot from '@lucide/svelte/icons/circle-dot'
  import CheckCircle2 from '@lucide/svelte/icons/check-circle-2'
  import Ban from '@lucide/svelte/icons/ban'
  import { dependencyRelationActions, type DependencyDraft } from '../mutation-shared'
  import type { TicketDetail, TicketReferenceOption } from '../types'
  import { cn } from '$lib/utils'

  let {
    ticket,
    availableTickets,
    creatingDependency = false,
    deletingDependencyId = null,
    onAddDependency,
    onDeleteDependency,
  }: {
    ticket: TicketDetail
    availableTickets: TicketReferenceOption[]
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    onAddDependency?: (draft: DependencyDraft) => Promise<boolean> | boolean
    onDeleteDependency?: (dependencyId: string) => void
  } = $props()

  let popoverOpen = $state(false)
  let selectedRelation = $state<DependencyDraft['relation'] | null>(null)
  let search = $state('')

  const dependencyRelationLabels: Record<string, string> = {
    blocked_by: 'Blocked by',
    blocks: 'Blocks',
    sub_issue: 'Sub-issue',
  }

  const dependencyOptions = $derived.by(() =>
    availableTickets.filter(
      (candidate) =>
        !ticket.dependencies.some((dependency) => dependency.targetId === candidate.id),
    ),
  )

  const filteredTickets = $derived.by(() => {
    const q = search.trim().toLowerCase()
    if (!q) return dependencyOptions
    return dependencyOptions.filter(
      (t) => t.identifier.toLowerCase().includes(q) || t.title.toLowerCase().includes(q),
    )
  })

  const dependencyGroups = $derived.by(() => {
    const blockedBy = ticket.dependencies.filter(
      (dependency) => dependency.relation === 'blocked_by',
    )
    const blocking = ticket.dependencies.filter((dependency) => dependency.relation === 'blocks')
    const hierarchy = ticket.dependencies.filter(
      (dependency) => dependency.relation === 'sub_issue',
    )

    return [
      { key: 'blocked_by', title: 'Blocked by', items: blockedBy },
      { key: 'blocks', title: 'Blocking', items: blocking },
      { key: 'sub_issue', title: 'Hierarchy', items: hierarchy },
    ].filter((group) => group.items.length > 0)
  })

  function resetPopover() {
    selectedRelation = null
    search = ''
  }

  function handleOpenChange(open: boolean) {
    popoverOpen = open
    if (!open) resetPopover()
  }

  async function handleSelectTicket(ticketId: string) {
    if (!selectedRelation || creatingDependency) return
    const accepted =
      (await onAddDependency?.({ targetTicketId: ticketId, relation: selectedRelation })) ?? false
    if (accepted) {
      popoverOpen = false
      resetPopover()
    }
  }
</script>

<section class="space-y-3">
  <div class="flex items-center justify-between gap-3">
    <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
      Dependencies
    </span>

    <Popover.Root open={popoverOpen} onOpenChange={handleOpenChange}>
      <Popover.Trigger
        class={buttonVariants({ variant: 'outline', size: 'icon-sm' })}
        aria-label="Add dependency"
        disabled={!dependencyOptions.length}
      >
        <Plus class="size-3.5" />
      </Popover.Trigger>
      <Popover.Content align="end" class="w-72 p-0">
        {#if !selectedRelation}
          <!-- Step 1: Choose relation type -->
          <div class="p-1">
            {#each dependencyRelationActions as action (action.relation)}
              <button
                type="button"
                class="hover:bg-muted flex w-full items-start gap-3 rounded-md px-3 py-2 text-left transition-colors"
                onclick={() => (selectedRelation = action.relation)}
              >
                <div class="min-w-0">
                  <div class="text-foreground text-sm font-medium">{action.label}</div>
                  <div class="text-muted-foreground text-xs">{action.description}</div>
                </div>
              </button>
            {/each}
          </div>
        {:else}
          <!-- Step 2: Search and select ticket -->
          <Command.Root shouldFilter={false}>
            <div class="border-border flex items-center gap-1.5 border-b px-2 py-1.5">
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground shrink-0 rounded p-0.5 transition-colors"
                onclick={() => {
                  selectedRelation = null
                  search = ''
                }}
                aria-label="Back"
              >
                <ArrowLeft class="size-3.5" />
              </button>
              <Command.Input
                placeholder="Search tickets…"
                class="text-foreground placeholder:text-muted-foreground h-8 flex-1 border-0 bg-transparent text-sm shadow-none outline-none focus:ring-0"
                bind:value={search}
              />
            </div>
            <Command.List class="max-h-52 overflow-y-auto p-1">
              {#if creatingDependency}
                <div class="text-muted-foreground px-3 py-4 text-center text-xs">Adding…</div>
              {:else if filteredTickets.length === 0}
                <Command.Empty class="text-muted-foreground px-3 py-4 text-center text-xs">
                  No matching tickets.
                </Command.Empty>
              {:else}
                {#each filteredTickets as option (option.id)}
                  <Command.Item
                    value={option.id}
                    class="flex items-center gap-2 rounded-md px-2 py-1.5 text-xs"
                    onSelect={() => void handleSelectTicket(option.id)}
                  >
                    <span class="text-muted-foreground shrink-0 font-mono">{option.identifier}</span
                    >
                    <span class="truncate">{option.title}</span>
                  </Command.Item>
                {/each}
              {/if}
            </Command.List>
          </Command.Root>
        {/if}
      </Popover.Content>
    </Popover.Root>
  </div>

  {#if dependencyGroups.length > 0}
    <div class="space-y-2">
      {#each dependencyGroups as group (group.key)}
        <div class="space-y-2">
          <div class="text-muted-foreground px-1 text-[10px] font-medium tracking-wider uppercase">
            {group.title}
          </div>
          {#each group.items as dependency (dependency.id)}
            {@const isTerminal =
              dependency.stage === 'completed' || dependency.stage === 'canceled'}
            {@const isBlocked = dependency.relation === 'blocked_by' && !isTerminal}
            <div
              class="border-border bg-muted/20 flex items-center gap-3 rounded-md border px-3 py-2"
            >
              <div class="shrink-0">
                {#if isBlocked}
                  <Ban class="size-4 text-red-500" />
                {:else if isTerminal}
                  <CheckCircle2 class="size-4 text-purple-500" />
                {:else}
                  <CircleDot class="size-4 text-green-500" />
                {/if}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span
                    class={cn(
                      'font-mono text-[11px]',
                      isTerminal ? 'text-muted-foreground line-through' : 'text-muted-foreground',
                    )}
                  >
                    {dependency.identifier}
                  </span>
                  <Badge variant="outline" class="h-4 py-0 text-[10px]">
                    {dependencyRelationLabels[dependency.relation] ?? dependency.relation}
                  </Badge>
                  {#if isBlocked}
                    <Badge
                      variant="outline"
                      class="h-4 border-red-500/30 bg-red-500/10 py-0 text-[10px] text-red-500"
                    >
                      Blocked
                    </Badge>
                  {/if}
                </div>
                <p
                  class={cn(
                    'mt-1 text-xs leading-4',
                    isTerminal ? 'text-muted-foreground' : 'text-foreground',
                  )}
                >
                  {dependency.title}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon-sm"
                disabled={deletingDependencyId === dependency.id}
                onclick={() => onDeleteDependency?.(dependency.id)}
                aria-label={`Remove ${dependency.identifier} relationship`}
              >
                <Trash2 class="size-3.5" />
              </Button>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  {/if}
</section>
