<script lang="ts">
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { dependencyRelationOptions } from '../mutation-shared'
  import type { TicketReferenceOption } from '../types'

  let {
    availableTickets,
    creating = false,
    onCreate,
    onCancel,
  }: {
    availableTickets: TicketReferenceOption[]
    creating?: boolean
    onCreate?: (draft: { targetTicketId: string; relation: string }) => Promise<boolean> | boolean
    onCancel?: () => void
  } = $props()

  let dependencyTargetId = $state('')
  let dependencyRelation = $state<string>(dependencyRelationOptions[0]?.value ?? 'blocks')

  const selectedDependencyOption = $derived(
    availableTickets.find((option) => option.id === dependencyTargetId) ?? null,
  )

  $effect(() => {
    if (!availableTickets.length) {
      dependencyTargetId = ''
      return
    }
    if (!availableTickets.some((option) => option.id === dependencyTargetId)) {
      dependencyTargetId = availableTickets[0]?.id ?? ''
    }
  })

  async function handleCreate() {
    const accepted =
      (await onCreate?.({
        targetTicketId: dependencyTargetId,
        relation: dependencyRelation,
      })) ?? false

    if (accepted) {
      dependencyRelation = dependencyRelationOptions[0]?.value ?? 'blocks'
    }
  }
</script>

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
        {#each availableTickets as option (option.id)}
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

  <div class="flex justify-end gap-2">
    {#if onCancel}
      <Button size="sm" variant="outline" onclick={onCancel} disabled={creating}>Cancel</Button>
    {/if}
    <Button
      size="sm"
      onclick={handleCreate}
      disabled={!availableTickets.length || !dependencyTargetId || creating}
    >
      {creating ? 'Adding…' : 'Add dependency'}
    </Button>
  </div>
</div>
