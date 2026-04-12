<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import type { ChannelDraft } from '../notification-channels'
  import NotificationChannelEditor from './notification-channel-editor.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      <Dialog.Title>
        {i18nStore.t(
          editingChannel
            ? 'settings.notificationChannel.dialogs.title.edit'
            : 'settings.notificationChannel.dialogs.title.create',
        )}
      </Dialog.Title>
      {#if editingChannel}
        <Dialog.Description>{editingChannel.name}</Dialog.Description>
      {:else}
        <Dialog.Description>
          {i18nStore.t('settings.notificationChannel.dialogs.description.create')}
        </Dialog.Description>
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
          {i18nStore.t('settings.notificationChannel.dialogs.buttons.delete')}
        </Button>
      {/if}
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={saving || deleting}>
            {i18nStore.t('settings.notificationChannel.dialogs.buttons.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button onclick={onSave} disabled={saving || deleting}>
        {saving
          ? i18nStore.t('settings.notificationChannel.dialogs.buttons.saving')
          : editingChannel
            ? i18nStore.t('settings.notificationChannel.dialogs.buttons.saveChanges')
            : i18nStore.t('settings.notificationChannel.dialogs.buttons.create')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={confirmDeleteOpen} onOpenChange={onConfirmDeleteOpenChange}>
  <Dialog.Content class="sm:max-w-sm">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('settings.notificationChannel.dialogs.confirm.title', {
          name: editingChannel?.name ?? '',
        })}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('settings.notificationChannel.dialogs.confirm.description')}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>
            {i18nStore.t('settings.notificationChannel.dialogs.buttons.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={onDelete} disabled={deleting}>
        {deleting
          ? i18nStore.t('settings.notificationChannel.dialogs.buttons.deleting')
          : i18nStore.t('settings.notificationChannel.dialogs.buttons.delete')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
