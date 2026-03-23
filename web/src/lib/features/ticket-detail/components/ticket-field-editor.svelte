<script lang="ts">
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import type { TicketDetail } from '../types'

  let {
    ticket,
    saving = false,
    onSave,
  }: {
    ticket: TicketDetail
    saving?: boolean
    onSave?: (draft: { title: string; description: string; statusId: string }) => void
  } = $props()

  let descriptionDraft = $derived(ticket.description)
  const fieldsDirty = $derived(descriptionDraft.trim() !== ticket.description)

  function handleSave() {
    onSave?.({
      title: ticket.title,
      description: descriptionDraft,
      statusId: ticket.status.id,
    })
  }
</script>

<section class="border-border bg-muted/20 rounded-lg border p-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-sm font-medium">Description</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Update the issue description without leaving the drawer.
      </p>
    </div>
    <Button size="sm" onclick={handleSave} disabled={!fieldsDirty || saving}>
      {saving ? 'Saving…' : 'Save description'}
    </Button>
  </div>

  <div class="mt-4">
    <div class="space-y-2">
      <Label for="ticket-description">Description</Label>
      <Textarea id="ticket-description" rows={5} bind:value={descriptionDraft} disabled={saving} />
    </div>
  </div>
</section>
