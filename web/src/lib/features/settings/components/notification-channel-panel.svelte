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
  import NotificationChannelEditor from './notification-channel-editor.svelte'

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
  let creatingNew = $state(false)
  let draft = $state<ChannelDraft>(createChannelDraft())
  let saving = $state(false)
  let deleting = $state(false)
  let testing = $state(false)
  let togglingId = $state<string | null>(null)

  function openNew() {
    editingChannel = null
    creatingNew = true
    draft = createChannelDraft()
  }

  function openEdit(channel: NotificationChannel) {
    editingChannel = channel
    creatingNew = true
    draft = channelDraftFromRecord(channel)
  }

  function closeEditor() {
    editingChannel = null
    creatingNew = false
  }

  async function handleSave() {
    if (editingChannel) {
      const parsed = buildUpdateChannelInput(draft, editingChannel)
      if (!parsed.ok) {
        toastStore.error(parsed.error)
        return
      }
      if (!parsed.value.changed) {
        toastStore.info('No channel changes to save.')
        return
      }

      saving = true
      try {
        await onUpdate(editingChannel.id, parsed.value.value)
        toastStore.success('Channel updated.')
        closeEditor()
      } catch (caughtError) {
        toastStore.error(actionErrorMessage(caughtError, 'Failed to update channel.'))
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
      toastStore.success('Channel created.')
      closeEditor()
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to create channel.'))
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!editingChannel) return
    if (!window.confirm(`Delete channel "${editingChannel.name}"?`)) return

    deleting = true
    try {
      await onDelete(editingChannel.id)
      toastStore.success('Channel deleted.')
      closeEditor()
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to delete channel.'))
    } finally {
      deleting = false
    }
  }

  async function handleToggle(channel: NotificationChannel) {
    togglingId = channel.id
    try {
      const updated = await onToggle(channel.id, !channel.is_enabled)
      toastStore.success(updated.is_enabled ? 'Channel enabled.' : 'Channel disabled.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to update channel state.'))
    } finally {
      togglingId = null
    }
  }

  async function handleTest(channel: NotificationChannel) {
    testing = true
    try {
      await onTest(channel.id)
      toastStore.success('Test notification sent.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to send test notification.'))
    } finally {
      testing = false
    }
  }

  const channelTypeLabels: Record<string, string> = {
    webhook: 'Webhook',
    telegram: 'Telegram',
    slack: 'Slack',
    wecom: 'WeCom',
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Channels</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Organization-level delivery endpoints shared across projects.
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={openNew}>Add channel</Button>
  </div>

  {#if channels.length === 0 && !creatingNew}
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
      <p class="text-muted-foreground text-sm">No channels configured.</p>
      <p class="text-muted-foreground text-xs">
        Add a channel to start receiving notifications via Webhook, Slack, Telegram, or WeCom.
      </p>
      <Button variant="outline" size="sm" class="mt-2" onclick={openNew}>Add channel</Button>
    </div>
  {/if}

  {#if channels.length > 0}
    <div class="grid gap-3 sm:grid-cols-2">
      {#each channels as channel (channel.id)}
        <div
          class="border-border bg-card group rounded-lg border px-4 py-3 transition-colors hover:border-border"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="truncate text-sm font-medium">{channel.name}</span>
                <Badge variant="outline" class="shrink-0 text-[10px] uppercase">
                  {channelTypeLabels[channel.type] ?? channel.type}
                </Badge>
              </div>
              <p class="text-muted-foreground mt-1 text-xs">
                {channel.is_enabled ? 'Receiving notifications' : 'Paused'}
              </p>
            </div>
            <Switch
              checked={channel.is_enabled}
              disabled={togglingId === channel.id}
              onCheckedChange={() => handleToggle(channel)}
            />
          </div>
          <div class="mt-3 flex items-center gap-1.5 border-t border-border/50 pt-3">
            <Button variant="ghost" size="sm" class="h-7 px-2 text-xs" onclick={() => openEdit(channel)}>
              Edit
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="h-7 px-2 text-xs"
              disabled={testing}
              onclick={() => handleTest(channel)}
            >
              {testing ? 'Sending...' : 'Send test'}
            </Button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if creatingNew}
    <div class="border-border bg-card rounded-lg border">
      <div class="flex items-center justify-between border-b border-border/50 px-5 py-3">
        <h4 class="text-sm font-medium">
          {editingChannel ? `Edit: ${editingChannel.name}` : 'New channel'}
        </h4>
        <Button variant="ghost" size="sm" class="h-7 px-2 text-xs" onclick={closeEditor}>
          Cancel
        </Button>
      </div>
      <NotificationChannelEditor
        {draft}
        selectedChannel={editingChannel}
        {saving}
        {deleting}
        onDraftChange={(nextDraft) => {
          draft = nextDraft
        }}
        onSave={handleSave}
        onDelete={handleDelete}
      />
    </div>
  {/if}
</div>
