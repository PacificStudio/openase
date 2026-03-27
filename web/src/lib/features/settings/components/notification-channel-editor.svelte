<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { applyChannelTypeTemplate, type ChannelDraft } from '../notification-channels'

  let {
    draft,
    selectedChannel,
    saving = false,
    deleting = false,
    testing = false,
    toggling = false,
    onDraftChange,
    onSave,
    onDelete,
    onToggle,
    onTest,
  }: {
    draft: ChannelDraft
    selectedChannel: NotificationChannel | null
    saving?: boolean
    deleting?: boolean
    testing?: boolean
    toggling?: boolean
    onDraftChange: (draft: ChannelDraft) => void
    onSave: () => void
    onDelete: () => void
    onToggle: () => void
    onTest: () => void
  } = $props()

  function updateTextField(field: 'name' | 'configText', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange({ ...draft, [field]: target.value })
  }
</script>

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
          onclick={onToggle}
          disabled={saving || deleting || testing || toggling}
        >
          {toggling ? 'Updating…' : selectedChannel.is_enabled ? 'Disable' : 'Enable'}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onclick={onTest}
          disabled={saving || deleting || testing || toggling}
        >
          {testing ? 'Sending…' : 'Send test'}
        </Button>
      {/if}
      <Button size="sm" onclick={onSave} disabled={saving || deleting || testing || toggling}>
        {saving ? 'Saving…' : selectedChannel ? 'Save changes' : 'Create channel'}
      </Button>
      {#if selectedChannel}
        <Button
          variant="destructive"
          size="sm"
          onclick={onDelete}
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
          onDraftChange(applyChannelTypeTemplate(draft, value || 'webhook'))
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
      Webhook expects `url`, optional `headers`, and optional `secret`. Slack expects `webhook_url`.
      Telegram expects `bot_token` and `chat_id`. WeCom expects `webhook_key`.
    </p>
  </div>
</div>
