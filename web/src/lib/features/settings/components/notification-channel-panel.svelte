<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Switch } from '$ui/switch'
  import {
    buildCreateChannelInput,
    buildUpdateChannelInput,
    channelDraftFromRecord,
    createChannelDraft,
    type ChannelCreateInput,
    type ChannelDraft,
    type ChannelUpdateInput,
  } from '../notification-channels'
  import { actionErrorMessage } from '../notification-support'
  import { toastStore } from '$lib/stores/toast.svelte'
  import NotificationChannelDialog from './notification-channel-dialog.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { TranslationKey } from '$lib/i18n'

  let {
    channels,
    onCreate,
    onUpdate,
    onDelete,
    onToggle,
    onTest,
  }: {
    channels: NotificationChannel[]
    onCreate: (input: ChannelCreateInput) => Promise<NotificationChannel>
    onUpdate: (channelId: string, input: ChannelUpdateInput) => Promise<NotificationChannel>
    onDelete: (channelId: string) => Promise<void>
    onToggle: (channelId: string, isEnabled: boolean) => Promise<NotificationChannel>
    onTest: (channelId: string) => Promise<void>
  } = $props()

  let editingChannel = $state<NotificationChannel | null>(null)
  let draft = $state<ChannelDraft>(createChannelDraft())
  let dialogOpen = $state(false)
  let confirmDeleteOpen = $state(false)
  let saving = $state(false)
  let deleting = $state(false)
  let testing = $state(false)
  let togglingId = $state<string | null>(null)

  function openNew() {
    editingChannel = null
    draft = createChannelDraft()
    dialogOpen = true
  }

  function openEdit(channel: NotificationChannel) {
    editingChannel = channel
    draft = channelDraftFromRecord(channel)
    dialogOpen = true
  }

  function closeDialog() {
    if (saving || deleting) return
    dialogOpen = false
    editingChannel = null
  }

  async function handleSave() {
    if (editingChannel) {
      const parsed = buildUpdateChannelInput(draft, editingChannel)
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return
      }
      if (!parsed.value.changed) {
        toastStore.info(i18nStore.t('settings.notificationChannel.panel.messages.noChanges'))
        return
      }
      saving = true
      try {
        await onUpdate(editingChannel.id, parsed.value.value)
        toastStore.success(i18nStore.t('settings.notificationChannel.panel.messages.updated'))
        dialogOpen = false
        editingChannel = null
      } catch (caughtError) {
        toastStore.error(
          actionErrorMessage(
            caughtError,
            i18nStore.t('settings.notificationChannel.panel.errors.update'),
          ),
        )
      } finally {
        saving = false
      }
      return
    }

    const parsed = buildCreateChannelInput(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }
    saving = true
    try {
      await onCreate(parsed.value)
      toastStore.success(i18nStore.t('settings.notificationChannel.panel.messages.created'))
      dialogOpen = false
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(
          caughtError,
          i18nStore.t('settings.notificationChannel.panel.errors.create'),
        ),
      )
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!editingChannel) return
    deleting = true
    try {
      await onDelete(editingChannel.id)
      toastStore.success(i18nStore.t('settings.notificationChannel.panel.messages.deleted'))
      dialogOpen = false
      confirmDeleteOpen = false
      editingChannel = null
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(
          caughtError,
          i18nStore.t('settings.notificationChannel.panel.errors.delete'),
        ),
      )
    } finally {
      deleting = false
    }
  }

  async function handleToggle(channel: NotificationChannel) {
    togglingId = channel.id
    try {
      const updated = await onToggle(channel.id, !channel.is_enabled)
      toastStore.success(
        i18nStore.t(
          updated.is_enabled
            ? NOTIFICATION_CHANNEL_TOGGLE_MESSAGES.enabled
            : NOTIFICATION_CHANNEL_TOGGLE_MESSAGES.disabled,
        ),
      )
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(
          caughtError,
          i18nStore.t('settings.notificationChannel.panel.errors.state'),
        ),
      )
    } finally {
      togglingId = null
    }
  }

  async function handleTest(channel: NotificationChannel) {
    testing = true
    try {
      await onTest(channel.id)
      toastStore.success(i18nStore.t('settings.notificationChannel.panel.messages.testSent'))
    } catch (caughtError) {
      toastStore.error(
        actionErrorMessage(
          caughtError,
          i18nStore.t('settings.notificationChannel.panel.errors.test'),
        ),
      )
    } finally {
      testing = false
    }
  }

  const CHANNEL_TYPE_LABEL_KEYS: Record<NotificationChannel['type'], TranslationKey> = {
    webhook: 'settings.notificationChannel.types.webhook',
    telegram: 'settings.notificationChannel.types.telegram',
    slack: 'settings.notificationChannel.types.slack',
    wecom: 'settings.notificationChannel.types.wecom',
  }

  const NOTIFICATION_CHANNEL_TOGGLE_MESSAGES: Record<'enabled' | 'disabled', TranslationKey> = {
    enabled: 'settings.notificationChannel.panel.messages.enabled',
    disabled: 'settings.notificationChannel.panel.messages.disabled',
  }

  const channelTypeLabel = (type: NotificationChannel['type']) =>
    i18nStore.t(CHANNEL_TYPE_LABEL_KEYS[type])
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.notificationChannel.panel.labels.heading')}
      </h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {i18nStore.t('settings.notificationChannel.panel.labels.description')}
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={openNew}>
      {i18nStore.t('settings.notificationChannel.panel.actions.add')}
    </Button>
  </div>

  {#if channels.length === 0}
    <div
      class="border-border bg-muted/30 flex flex-col items-center gap-2 rounded-lg border border-dashed px-6 py-8 text-center"
    >
      <svg
        class="text-muted-foreground size-8"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        stroke-width="1.5"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"
        />
      </svg>
      <p class="text-muted-foreground text-sm">
        {i18nStore.t('settings.notificationChannel.panel.messages.noChannels')}
      </p>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('settings.notificationChannel.panel.messages.addHint')}
      </p>
      <Button variant="outline" size="sm" class="mt-2" onclick={openNew}>
        {i18nStore.t('settings.notificationChannel.panel.actions.add')}
      </Button>
    </div>
  {:else}
    <div class="grid gap-3 sm:grid-cols-2">
      {#each channels as channel (channel.id)}
        <div class="border-border bg-card rounded-lg border px-4 py-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="truncate text-sm font-medium">{channel.name}</span>
                <Badge variant="outline" class="shrink-0 text-[10px] uppercase">
                  {channelTypeLabel(channel.type)}
                </Badge>
              </div>
              <p class="text-muted-foreground mt-1 text-xs">
                {channel.is_enabled
                  ? i18nStore.t('settings.notificationChannel.panel.labels.enabled')
                  : i18nStore.t('settings.notificationChannel.panel.labels.paused')}
              </p>
            </div>
            <Switch
              checked={channel.is_enabled}
              disabled={togglingId === channel.id}
              onCheckedChange={() => handleToggle(channel)}
            />
          </div>
          <div class="border-border/50 mt-3 flex items-center gap-1.5 border-t pt-3">
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              onclick={() => openEdit(channel)}
            >
              {i18nStore.t('settings.notificationChannel.panel.actions.edit')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              disabled={testing}
              onclick={() => handleTest(channel)}
            >
              {testing
                ? i18nStore.t('settings.notificationChannel.panel.actions.sending')
                : i18nStore.t('settings.notificationChannel.panel.actions.sendTest')}
            </Button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<NotificationChannelDialog
  {editingChannel}
  {draft}
  {dialogOpen}
  {confirmDeleteOpen}
  {saving}
  {deleting}
  onDialogOpenChange={(open) => {
    if (!open) closeDialog()
    else dialogOpen = true
  }}
  onConfirmDeleteOpenChange={(open) => (confirmDeleteOpen = open)}
  onDraftChange={(nextDraft) => {
    draft = nextDraft
  }}
  onSave={handleSave}
  onDelete={handleDelete}
/>
