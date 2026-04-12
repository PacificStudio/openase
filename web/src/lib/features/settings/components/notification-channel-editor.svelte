<script lang="ts">
  import type { NotificationChannel } from '$lib/api/contracts'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { applyChannelTypeTemplate, type ChannelDraft } from '../notification-channels'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { TranslationKey } from '$lib/i18n'

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

  const CHANNEL_TYPE_LABEL_KEYS: Record<ChannelDraft['type'], TranslationKey> = {
    webhook: 'settings.notificationChannel.types.webhook',
    telegram: 'settings.notificationChannel.types.telegram',
    slack: 'settings.notificationChannel.types.slack',
    wecom: 'settings.notificationChannel.types.wecom',
  }

  const translateChannelType = (type: ChannelDraft['type']) =>
    i18nStore.t(CHANNEL_TYPE_LABEL_KEYS[type])
  const configKeys = {
    url: /* i18n-exempt */ 'url',
    headers: /* i18n-exempt */ 'headers',
    secret: /* i18n-exempt */ 'secret',
    webhook_url: /* i18n-exempt */ 'webhook_url',
    bot_token: /* i18n-exempt */ 'bot_token',
    chat_id: /* i18n-exempt */ 'chat_id',
    webhook_key: /* i18n-exempt */ 'webhook_key',
  }
</script>

<div class="space-y-4">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="notification-channel-name">{i18nStore.t('settings.notificationChannel.labels.name')}</Label>
      <Input
        id="notification-channel-name"
        placeholder={i18nStore.t('settings.notificationChannel.placeholders.nameExample')}
        value={draft.name}
        oninput={(event) => updateTextField('name', event)}
      />
    </div>

    <div class="space-y-1.5">
      <Label>{i18nStore.t('settings.notificationChannel.labels.type')}</Label>
      <Select.Root
        type="single"
        value={draft.type}
        onValueChange={(value) => {
          onDraftChange(applyChannelTypeTemplate(draft, value || 'webhook'))
        }}
      >
        <Select.Trigger class="w-full">
          <span class="uppercase">{translateChannelType(draft.type)}</span>
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="webhook">{i18nStore.t('settings.notificationChannel.types.webhook')}</Select.Item>
          <Select.Item value="telegram">{i18nStore.t('settings.notificationChannel.types.telegram')}</Select.Item>
          <Select.Item value="slack">{i18nStore.t('settings.notificationChannel.types.slack')}</Select.Item>
          <Select.Item value="wecom">{i18nStore.t('settings.notificationChannel.types.wecom')}</Select.Item>
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for="notification-channel-config">{i18nStore.t('settings.notificationChannel.labels.configuration')}</Label>
    <Textarea
      id="notification-channel-config"
      value={draft.configText}
      rows={8}
      class="font-mono text-xs"
      oninput={(event) => updateTextField('configText', event)}
    />
    <p class="text-muted-foreground text-xs">
      {#if draft.type === 'webhook'}
        <span>{i18nStore.t('settings.notificationChannel.hints.requires')}</span>
        <code class="bg-muted rounded px-1">{configKeys.url}</code>.{' '}
        <span>{i18nStore.t('settings.notificationChannel.hints.optional')}</span>{' '}
        <code class="bg-muted rounded px-1">{configKeys.headers}</code>,{' '}
        <code class="bg-muted rounded px-1">{configKeys.secret}</code>.
      {:else if draft.type === 'slack'}
        <span>{i18nStore.t('settings.notificationChannel.hints.requires')}</span>{' '}
        <code class="bg-muted rounded px-1">{configKeys.webhook_url}</code>.
      {:else if draft.type === 'telegram'}
        <span>{i18nStore.t('settings.notificationChannel.hints.requires')}</span>{' '}
        <code class="bg-muted rounded px-1">{configKeys.bot_token}</code>{' '}
        <span>{i18nStore.t('settings.notificationChannel.hints.and')}</span>{' '}
        <code class="bg-muted rounded px-1">{configKeys.chat_id}</code>.
      {:else if draft.type === 'wecom'}
        <span>{i18nStore.t('settings.notificationChannel.hints.requires')}</span>{' '}
        <code class="bg-muted rounded px-1">{configKeys.webhook_key}</code>.
      {:else}
        {i18nStore.t('settings.notificationChannel.hints.validJson')}
      {/if}
      {#if selectedChannel}
        {i18nStore.t('settings.notificationChannel.hints.rotation')}
      {/if}
    </p>
  </div>
</div>
