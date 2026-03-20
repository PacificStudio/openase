<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import {
    applyChannelTypeTemplate,
    buildCreateChannelInput,
    buildUpdateChannelInput,
    channelDraftFromRecord,
    createChannelDraft,
    type ChannelCreateInput,
    type ChannelDraft,
    type ChannelUpdateInput,
  } from '../notification-channels'

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
  let feedback = $state('')
  let error = $state('')

  const selectedChannel = $derived(channels.find((channel) => channel.id === selectedId) ?? null)

  $effect(() => {
    if (selectedId !== 'new' && !selectedChannel) {
      selectedId = channels[0]?.id ?? 'new'
    }

    draft = selectedChannel ? channelDraftFromRecord(selectedChannel) : createChannelDraft()
  })

  function selectChannel(channelId: string) {
    selectedId = channelId
    feedback = ''
    error = ''
  }

  function selectNewChannel() {
    selectedId = 'new'
    feedback = ''
    error = ''
  }

  function updateTextField(field: 'name' | 'configText', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    draft = { ...draft, [field]: target.value }
  }

  async function handleSave() {
    feedback = ''
    error = ''

    if (selectedChannel) {
      const parsed = buildUpdateChannelInput(draft, selectedChannel)
      if (!parsed.ok) {
        error = parsed.error
        return
      }
      if (!parsed.value.changed) {
        feedback = 'No channel changes to save.'
        return
      }

      saving = true
      try {
        const channel = await onUpdate(selectedChannel.id, parsed.value.value)
        selectedId = channel.id
        feedback = 'Channel updated.'
      } finally {
        saving = false
      }
      return
    }

    const parsed = buildCreateChannelInput(draft)
    if (!parsed.ok) {
      error = parsed.error
      return
    }

    saving = true
    try {
      const channel = await onCreate(parsed.value)
      selectedId = channel.id
      feedback = 'Channel created.'
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (!selectedChannel) return
    if (!window.confirm(`Delete channel "${selectedChannel.name}"?`)) return

    feedback = ''
    error = ''
    deleting = true
    try {
      await onDelete(selectedChannel.id)
      selectedId = 'new'
      feedback = 'Channel deleted.'
    } finally {
      deleting = false
    }
  }

  async function handleToggle() {
    if (!selectedChannel) return

    feedback = ''
    error = ''
    toggling = true
    try {
      const channel = await onToggle(selectedChannel.id, !selectedChannel.is_enabled)
      selectedId = channel.id
      feedback = channel.is_enabled ? 'Channel enabled.' : 'Channel disabled.'
    } finally {
      toggling = false
    }
  }

  async function handleTest() {
    if (!selectedChannel) return

    feedback = ''
    error = ''
    testing = true
    try {
      await onTest(selectedChannel.id)
      feedback = 'Test notification sent.'
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
          <button type="button" class={`w-full rounded-xl border px-3 py-3 text-left transition-colors ${
            selectedId === channel.id ? 'border-primary/40 bg-primary/5' : 'border-border hover:bg-muted/50'
          }`} onclick={() => selectChannel(channel.id)}>
            <div class="flex items-center justify-between gap-2">
              <span class="text-sm font-medium">{channel.name}</span>
              <Badge variant="outline">{channel.is_enabled ? 'Enabled' : 'Disabled'}</Badge>
            </div>
            <p class="text-muted-foreground mt-1 text-xs uppercase">{channel.type}</p>
          </button>
        {/each}
      {/if}
    </div>
    <div class="space-y-5 px-5 py-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h4 class="text-foreground text-sm font-semibold">
            {selectedChannel ? selectedChannel.name : 'Create channel'}
          </h4>
          <p class="text-muted-foreground mt-1 text-sm">
            Existing configs are response-safe and may contain masked secrets. Replace the JSON only
            when you intend to rotate or rewrite the config.
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          {#if selectedChannel}
            <Button
              variant="outline"
              size="sm"
              onclick={handleToggle}
              disabled={saving || deleting || testing || toggling}
            >
              {toggling ? 'Updating…' : selectedChannel.is_enabled ? 'Disable' : 'Enable'}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onclick={handleTest}
              disabled={saving || deleting || testing || toggling}
            >
              {testing ? 'Sending…' : 'Send test'}
            </Button>
          {/if}
          <Button size="sm" onclick={handleSave} disabled={saving || deleting || testing || toggling}>
            {saving ? 'Saving…' : selectedChannel ? 'Save changes' : 'Create channel'}
          </Button>
          {#if selectedChannel}
            <Button
              variant="destructive"
              size="sm"
              onclick={handleDelete}
              disabled={saving || deleting || testing || toggling}
            >
              {deleting ? 'Deleting…' : 'Delete'}
            </Button>
          {/if}
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label for="notification-channel-name">Channel name</Label>
          <Input
            id="notification-channel-name"
            value={draft.name}
            oninput={(event) => updateTextField('name', event)}
          />
        </div>

        <div class="space-y-2">
          <Label>Channel type</Label>
          <Select.Root
            type="single"
            value={draft.type}
            onValueChange={(value) => {
              draft = applyChannelTypeTemplate(draft, value || 'webhook')
            }}
          >
            <Select.Trigger class="w-full uppercase">{draft.type}</Select.Trigger>
            <Select.Content>
              <Select.Item value="webhook">webhook</Select.Item>
              <Select.Item value="telegram">telegram</Select.Item>
              <Select.Item value="slack">slack</Select.Item>
              <Select.Item value="wecom">wecom</Select.Item>
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      <div class="space-y-2">
        <Label for="notification-channel-config">Config JSON</Label>
        <Textarea
          id="notification-channel-config"
          value={draft.configText}
          rows={12}
          class="font-mono text-xs"
          oninput={(event) => updateTextField('configText', event)}
        />
        <p class="text-muted-foreground text-xs">
          Webhook expects `url`, optional `headers`, and optional `secret`. Slack expects
          `webhook_url`. Telegram expects `bot_token` and `chat_id`. WeCom expects `webhook_key`.
        </p>
      </div>

      {#if feedback}
        <p class="text-sm text-emerald-400">{feedback}</p>
      {/if}

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}
    </div>
  </div>
</div>
