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
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      <Dialog.Title>
        {i18nStore.t(
          editingRule
            ? 'settings.notificationRule.dialog.title.edit'
            : 'settings.notificationRule.dialog.title.create',
        )}
      </Dialog.Title>
      {#if editingRule}
        <Dialog.Description>{editingRule.name}</Dialog.Description>
      {:else}
        <Dialog.Description>
          {i18nStore.t('settings.notificationRule.dialog.description.create')}
        </Dialog.Description>
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
          {i18nStore.t('settings.notificationRule.dialog.buttons.delete')}
        </Button>
      {/if}
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={saving || deleting}>
            {i18nStore.t('settings.notificationRule.dialog.buttons.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSave} disabled={saving || deleting}>
        {saving
          ? i18nStore.t('settings.notificationRule.dialog.buttons.saving')
          : editingRule
            ? i18nStore.t('settings.notificationRule.dialog.buttons.saveChanges')
            : i18nStore.t('settings.notificationRule.dialog.buttons.create')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={confirmDeleteOpen} onOpenChange={onConfirmDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('settings.notificationRule.dialog.confirm.title', {
          name: editingRule?.name ?? '',
        })}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.notificationRule.dialog.confirm.description')}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>
            {i18nStore.t('settings.notificationRule.dialog.buttons.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={deleting}>
        {deleting
          ? i18nStore.t('settings.notificationRule.dialog.buttons.deleting')
          : i18nStore.t('settings.notificationRule.dialog.buttons.deleteRule')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
