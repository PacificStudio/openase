<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { applyChannelTypeTemplate, type ChannelDraft } from '../notification-channels'

  let {
    draft,
    selectedChannel,
    onDraftChange,
  }: {
    draft: ChannelDraft
    selectedChannel: NotificationChannel | null
    onDraftChange: (draft: ChannelDraft) => void
  } = $props()

  function updateTextField(field: 'name' | 'configText', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange({ ...draft, [field]: target.value })
  }
</script>

<div class="space-y-4">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="notification-channel-name">Name</Label>
      <Input
        id="notification-channel-name"
        placeholder="e.g. Ops Alerts"
        value={draft.name}
        oninput={(event) => updateTextField('name', event)}
      />
    </div>

    <div class="space-y-1.5">
      <Label>Type</Label>
      <Select.Root
        type="single"
        value={draft.type}
        onValueChange={(value) => {
          onDraftChange(applyChannelTypeTemplate(draft, value || 'webhook'))
        }}
      >
        <Select.Trigger class="w-full">
          <span class="uppercase">{draft.type}</span>
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="webhook">Webhook</Select.Item>
          <Select.Item value="telegram">Telegram</Select.Item>
          <Select.Item value="slack">Slack</Select.Item>
          <Select.Item value="wecom">WeCom</Select.Item>
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for="notification-channel-config">Configuration</Label>
    <Textarea
      id="notification-channel-config"
      value={draft.configText}
      rows={8}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('configText', event)}
    />
    <p class="text-muted-foreground text-xs">
      {#if draft.type === 'webhook'}
        Requires <code class="bg-muted rounded px-1">url</code>. Optional:
        <code class="bg-muted rounded px-1">headers</code>,
        <code class="bg-muted rounded px-1">secret</code>.
      {:else if draft.type === 'slack'}
        Requires <code class="bg-muted rounded px-1">webhook_url</code>.
      {:else if draft.type === 'telegram'}
        Requires <code class="bg-muted rounded px-1">bot_token</code> and
        <code class="bg-muted rounded px-1">chat_id</code>.
      {:else if draft.type === 'wecom'}
        Requires <code class="bg-muted rounded px-1">webhook_key</code>.
      {:else}
        Provide a valid JSON configuration object.
      {/if}
      {#if selectedChannel}
        Existing values may be masked. Replace only when rotating credentials.
      {/if}
    </p>
  </div>
</div>
