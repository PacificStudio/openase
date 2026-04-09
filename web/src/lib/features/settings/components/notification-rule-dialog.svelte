<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import type { RuleDraft } from '../notification-rules'
  import NotificationRuleEditor from './notification-rule-editor.svelte'

  let {
    channels,
    eventTypes,
    editingRule,
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
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    editingRule: NotificationRule | null
    draft: RuleDraft
    dialogOpen: boolean
    confirmDeleteOpen: boolean
    saving: boolean
    deleting: boolean
    onDialogOpenChange: (open: boolean) => void
    onConfirmDeleteOpenChange: (open: boolean) => void
    onDraftChange: (draft: RuleDraft) => void
    onSave: () => void
    onDelete: () => void
  } = $props()
</script>

<Dialog.Root open={dialogOpen} onOpenChange={onDialogOpenChange}>
  <Dialog.Content class="flex flex-col sm:max-w-2xl">
    <Dialog.Header>
      <Dialog.Title>{editingRule ? 'Edit rule' : 'New rule'}</Dialog.Title>
      {#if editingRule}
        <Dialog.Description>{editingRule.name}</Dialog.Description>
      {:else}
        <Dialog.Description>Route a project event to a notification channel.</Dialog.Description>
      {/if}
    </Dialog.Header>

    <div class="min-h-0 overflow-y-auto">
      <NotificationRuleEditor {channels} {draft} {eventTypes} {onDraftChange} />
    </div>

    <Dialog.Footer>
      {#if editingRule}
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
        {saving ? 'Saving…' : editingRule ? 'Save changes' : 'Create rule'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={confirmDeleteOpen} onOpenChange={onConfirmDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>Delete "{editingRule?.name}"?</Dialog.Title>
      <Dialog.Description>
        This rule will stop delivering notifications. This cannot be undone.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={deleting}>
        {deleting ? 'Deleting…' : 'Delete rule'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
