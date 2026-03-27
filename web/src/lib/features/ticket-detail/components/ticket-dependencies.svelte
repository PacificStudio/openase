<script lang="ts">
  import { Badge } from '$ui/badge'
  import { buttonVariants, Button } from '$ui/button'
  import * as Popover from '$ui/popover'
  import Plus from '@lucide/svelte/icons/plus'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import TicketDependencyForm from './ticket-dependency-form.svelte'
  import type { TicketDetail, TicketReferenceOption } from '../types'

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
    onAddDependency?: (draft: {
      targetTicketId: string
      relation: string
    }) => Promise<boolean> | boolean
    onDeleteDependency?: (dependencyId: string) => void
  } = $props()

  let createOpen = $state(false)

  const dependencyOptions = $derived.by(() =>
    availableTickets.filter(
      (candidate) =>
        !ticket.dependencies.some((dependency) => dependency.targetId === candidate.id),
    ),
  )

  async function handleCreate(draft: { targetTicketId: string; relation: string }) {
    const accepted = (await onAddDependency?.(draft)) ?? false
    if (accepted) {
      createOpen = false
    }
    return accepted
  }
</script>

<section class="space-y-3">
  <div class="flex items-start justify-between gap-3">
    <div>
      <h3 class="text-sm font-medium">Dependencies</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Add blockers or parent links directly from the current working context.
      </p>
    </div>

    <Popover.Root bind:open={createOpen}>
      <Popover.Trigger
        class={buttonVariants({ variant: 'outline', size: 'icon-sm' })}
        aria-label="Add dependency"
        disabled={!dependencyOptions.length}
      >
        <Plus class="size-3.5" />
      </Popover.Trigger>
      <Popover.Content align="end" class="w-80">
        <div class="space-y-3">
          <div>
            <h4 class="text-sm font-medium">Add dependency</h4>
            <p class="text-muted-foreground mt-1 text-xs">
              Select a related ticket and the dependency type.
            </p>
          </div>
          <TicketDependencyForm
            availableTickets={dependencyOptions}
            creating={creatingDependency}
            onCreate={handleCreate}
            onCancel={() => {
              createOpen = false
            }}
          />
        </div>
      </Popover.Content>
    </Popover.Root>
  </div>

  {#if ticket.dependencies.length === 0}
    <p class="text-muted-foreground rounded-md border border-dashed px-3 py-4 text-center text-xs">
      No dependencies linked yet.
    </p>
  {:else}
    <div class="space-y-2">
      {#each ticket.dependencies as dependency (dependency.id)}
        <div class="border-border bg-muted/20 flex items-center gap-3 rounded-md border px-3 py-2">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-muted-foreground font-mono text-[11px]">
                {dependency.identifier}
              </span>
              <Badge variant="outline" class="h-4 py-0 text-[10px]">
                {dependency.relation}
              </Badge>
            </div>
            <p class="text-foreground mt-1 truncate text-xs">{dependency.title}</p>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={deletingDependencyId === dependency.id}
            onclick={() => onDeleteDependency?.(dependency.id)}
          >
            <Trash2 class="size-3.5" />
          </Button>
        </div>
      {/each}
    </div>
  {/if}
</section>
