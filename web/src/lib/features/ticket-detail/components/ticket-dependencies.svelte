<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import { dependencyRelationOptions } from '../mutation-shared'
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
    onAddDependency?: (draft: { targetTicketId: string; relation: string }) => void
    onDeleteDependency?: (dependencyId: string) => void
  } = $props()

  let dependencyTargetId = $state('')
  let dependencyRelation = $state<string>(dependencyRelationOptions[0]?.value ?? 'blocks')

  const dependencyOptions = $derived.by(() =>
    availableTickets.filter(
      (candidate) =>
        !ticket.dependencies.some((dependency) => dependency.targetId === candidate.id),
    ),
  )
  const selectedDependencyOption = $derived(
    dependencyOptions.find((option) => option.id === dependencyTargetId) ?? null,
  )

  $effect(() => {
    if (!dependencyOptions.length) {
      dependencyTargetId = ''
      return
    }
    if (!dependencyOptions.some((option) => option.id === dependencyTargetId)) {
      dependencyTargetId = dependencyOptions[0]?.id ?? ''
    }
  })

  function handleAddDependency() {
    onAddDependency?.({
      targetTicketId: dependencyTargetId,
      relation: dependencyRelation,
    })
  }
</script>

<section class="space-y-3">
  <div>
    <h3 class="text-sm font-medium">Dependencies</h3>
    <p class="text-muted-foreground mt-1 text-xs">
      Add blockers or parent links directly from the current working context.
    </p>
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

  <div class="border-border bg-muted/20 rounded-lg border p-4">
    <div class="grid gap-3">
      <div class="space-y-2">
        <Label>Target ticket</Label>
        <Select.Root
          type="single"
          value={dependencyTargetId}
          onValueChange={(value) => {
            dependencyTargetId = value || ''
          }}
        >
          <Select.Trigger class="w-full">
            {selectedDependencyOption
              ? `${selectedDependencyOption.identifier} · ${selectedDependencyOption.title}`
              : 'Select a ticket'}
          </Select.Trigger>
          <Select.Content>
            {#each dependencyOptions as option (option.id)}
              <Select.Item value={option.id}>{option.identifier} · {option.title}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label>Relation</Label>
        <Select.Root
          type="single"
          value={dependencyRelation}
          onValueChange={(value) => {
            dependencyRelation = value || dependencyRelationOptions[0]?.value || 'blocks'
          }}
        >
          <Select.Trigger class="w-full">
            {dependencyRelationOptions.find((option) => option.value === dependencyRelation)?.label}
          </Select.Trigger>
          <Select.Content>
            {#each dependencyRelationOptions as option (option.value)}
              <Select.Item value={option.value}>{option.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="flex justify-end">
        <Button
          size="sm"
          onclick={handleAddDependency}
          disabled={!dependencyOptions.length || !dependencyTargetId || creatingDependency}
        >
          {creatingDependency ? 'Adding…' : 'Add dependency'}
        </Button>
      </div>
    </div>
  </div>
</section>
