<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import type { ChannelDraft } from '../notification-channels'
  import NotificationChannelEditor from './notification-channel-editor.svelte'

  let {
    editingChannel,
    draft,
    dialogOpen,
    confirmDeleteOpen,
    saving,
    deleting,
    onDialogOpenChange,
    onConfirmDeleteOpenChange,
    onDraftChange,
    onSave,
    onDelete,
  }: {
    editingChannel: NotificationChannel | null
    draft: ChannelDraft
    dialogOpen: boolean
    confirmDeleteOpen: boolean
    saving: boolean
    deleting: boolean
    onDialogOpenChange: (open: boolean) => void
    onConfirmDeleteOpenChange: (open: boolean) => void
    onDraftChange: (draft: ChannelDraft) => void
    onSave: () => void
    onDelete: () => void
  } = $props()
</script>

<Dialog.Root open={dialogOpen} onOpenChange={onDialogOpenChange}>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>{editingChannel ? 'Edit channel' : 'New channel'}</Dialog.Title>
      {#if editingChannel}
        <Dialog.Description>{editingChannel.name}</Dialog.Description>
      {:else}
        <Dialog.Description>Configure a new notification delivery endpoint.</Dialog.Description>
      {/if}
    </Dialog.Header>

    <div class="overflow-y-auto">
      <NotificationChannelEditor {draft} selectedChannel={editingChannel} {onDraftChange} />
    </div>

    <Dialog.Footer>
      {#if editingChannel}
        <Button
          variant="destructive"
          onclick={() => onConfirmDeleteOpenChange(true)}
          disabled={saving || deleting}
          class="mr-auto"
        >
          Delete
        </Button>
      {/if}
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={saving || deleting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSave} disabled={saving || deleting}>
        {saving ? 'Saving…' : editingChannel ? 'Save changes' : 'Create channel'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={confirmDeleteOpen} onOpenChange={onConfirmDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>Delete {editingChannel?.name}?</Dialog.Title>
      <Dialog.Description>
        This removes the channel and any rules that route to it. This cannot be undone.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={deleting}>
        {deleting ? 'Deleting…' : 'Delete channel'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
