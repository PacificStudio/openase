<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
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

  let selectedId = $state<string>('new')
  let draft = $state<ChannelDraft>(createChannelDraft())
  let saving = $state(false)
  let deleting = $state(false)
  let testing = $state(false)
  let toggling = $state(false)

  const selectedChannel = $derived(channels.find((channel) => channel.id === selectedId) ?? null)

  $effect(() => {
    if (selectedId !== 'new' && !selectedChannel) {
      selectedId = channels[0]?.id ?? 'new'
    }

    draft = selectedChannel ? channelDraftFromRecord(selectedChannel) : createChannelDraft()
  })

  function selectChannel(channelId: string) {
    selectedId = channelId
  }

  function selectNewChannel() {
    selectedId = 'new'
  }

  async function handleSave() {
    if (selectedChannel) {
      const parsed = buildUpdateChannelInput(draft, selectedChannel)
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
        const channel = await onUpdate(selectedChannel.id, parsed.value.value)
        selectedId = channel.id
        toastStore.success('Channel updated.')
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
      const channel = await onCreate(parsed.value)
      selectedId = channel.id
      toastStore.success('Channel created.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to create channel.'))
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!selectedChannel) return
    if (!window.confirm(`Delete channel "${selectedChannel.name}"?`)) return

    deleting = true
    try {
      await onDelete(selectedChannel.id)
      selectedId = 'new'
      toastStore.success('Channel deleted.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to delete channel.'))
    } finally {
      deleting = false
    }
  }

  async function handleToggle() {
    if (!selectedChannel) return

    toggling = true
    try {
      const channel = await onToggle(selectedChannel.id, !selectedChannel.is_enabled)
      selectedId = channel.id
      toastStore.success(channel.is_enabled ? 'Channel enabled.' : 'Channel disabled.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to update channel state.'))
    } finally {
      toggling = false
    }
  }

  async function handleTest() {
    if (!selectedChannel) return

    testing = true
    try {
      await onTest(selectedChannel.id)
      toastStore.success('Test notification sent.')
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to send test notification.'))
    } finally {
      testing = false
    }
  }
</script>

<div class="border-border bg-card rounded-2xl border">
  <div class="border-border flex items-start justify-between gap-4 border-b px-5 py-4">
    <div>
      <h3 class="text-foreground text-base font-semibold">Organization Channels</h3>
      <p class="text-muted-foreground mt-1 text-sm">
        Channels live at the org level and can be reused by notification rules across projects.
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={selectNewChannel}>New channel</Button>
  </div>
  <div class="grid gap-0 lg:grid-cols-[240px_minmax(0,1fr)]">
    <div class="border-border space-y-2 border-b px-4 py-4 lg:border-r lg:border-b-0">
      {#if channels.length === 0}
        <p class="text-muted-foreground text-sm">No channels yet.</p>
      {:else}
        {#each channels as channel (channel.id)}
          <button
            type="button"
            class={`w-full rounded-xl border px-3 py-3 text-left transition-colors ${
              selectedId === channel.id
                ? 'border-primary/40 bg-primary/5'
                : 'border-border hover:bg-muted/50'
            }`}
            onclick={() => selectChannel(channel.id)}
          >
            <div class="flex items-center justify-between gap-2">
              <span class="text-sm font-medium">{channel.name}</span>
              <Badge variant="outline">{channel.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
            </div>
            <p class="text-muted-foreground mt-1 text-xs uppercase">{channel.type}</p>
          </button>
        {/each}
      {/if}
    </div>
    <NotificationChannelEditor
      {draft}
      {selectedChannel}
      {saving}
      {deleting}
      {testing}
      {toggling}
      onDraftChange={(nextDraft) => {
        draft = nextDraft
      }}
      onSave={handleSave}
      onDelete={handleDelete}
      onToggle={handleToggle}
      onTest={handleTest}
    />
  </div>
</div>
