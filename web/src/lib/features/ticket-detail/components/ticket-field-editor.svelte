<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import type { TicketDetail, TicketStatusOption } from '../types'

  let {
    ticket,
    statuses,
    saving = false,
    onSave,
  }: {
    ticket: TicketDetail
    statuses: TicketStatusOption[]
    saving?: boolean
    onSave?: (draft: { title: string; description: string; statusId: string }) => void
  } = $props()

  let titleDraft = $state('')
  let descriptionDraft = $state('')
  let statusIdDraft = $state('')

  const fieldsDirty = $derived.by(
    () =>
      titleDraft.trim() !== ticket.title ||
      descriptionDraft.trim() !== ticket.description ||
      statusIdDraft !== ticket.status.id,
  )

  $effect(() => {
    titleDraft = ticket.title
    descriptionDraft = ticket.description
    statusIdDraft = ticket.status.id
  })

  function handleSave() {
    onSave?.({
      title: titleDraft,
      description: descriptionDraft,
      statusId: statusIdDraft,
    })
  }
</script>

<section class="border-border bg-muted/20 rounded-lg border p-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-sm font-medium">Editable Fields</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Update the ticket title, description, and current status without leaving the drawer.
      </p>
    </div>
    <Button size="sm" onclick={handleSave} disabled={!fieldsDirty || saving}>
      {saving ? 'Saving…' : 'Save fields'}
    </Button>
  </div>

  <div class="mt-4 space-y-4">
    <div class="space-y-2">
      <Label for="ticket-title">Title</Label>
      <Input id="ticket-title" bind:value={titleDraft} disabled={saving} />
    </div>

    <div class="space-y-2">
      <Label>Status</Label>
      <Select.Root
        type="single"
        value={statusIdDraft}
        onValueChange={(value) => {
          statusIdDraft = value || ticket.status.id
        }}
      >
        <Select.Trigger class="w-full">
          {statuses.find((status) => status.id === statusIdDraft)?.name ?? 'Select status'}
        </Select.Trigger>
        <Select.Content>
          {#each statuses as status (status.id)}
            <Select.Item value={status.id}>{status.name}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-2">
      <Label for="ticket-description">Description</Label>
      <Textarea id="ticket-description" rows={5} bind:value={descriptionDraft} disabled={saving} />
    </div>
  </div>
</section>
