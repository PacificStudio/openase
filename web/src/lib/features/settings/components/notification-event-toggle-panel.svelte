<script lang="ts">
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import * as Select from '$ui/select'
  import { Switch } from '$ui/switch'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { RuleCreateInput, RuleUpdateInput } from '../notification-rules'
  import {
    buildNotificationToggleRuleInput,
    findNotificationToggleRule,
    groupNotificationEventTypes,
  } from '../notification-event-toggles'
  import { actionErrorMessage } from '../notification-support'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { TranslationKey } from '$lib/i18n'

  let {
    channels,
    eventTypes,
    rules,
    onCreate,
    onDelete,
    onUpdate,
  }: {
    channels: NotificationChannel[]
    eventTypes: NotificationRuleEventType[]
    rules: NotificationRule[]
    onCreate: (input: RuleCreateInput) => Promise<NotificationRule>
    onDelete: (ruleId: string) => Promise<void>
    onUpdate: (ruleId: string, input: RuleUpdateInput) => Promise<NotificationRule>
  } = $props()

  let selectedChannelId = $state('')
  let actionKey = $state('')

  const availableChannels = $derived(channels.filter((channel) => channel.is_enabled))
  const groupedEvents = $derived(groupNotificationEventTypes(eventTypes))
  const NOTIFICATION_TOGGLE_MESSAGE_KEYS: Record<'enabled' | 'disabled', TranslationKey> = {
    enabled: 'settings.notificationEventToggle.notifications.enabled',
    disabled: 'settings.notificationEventToggle.notifications.disabled',
  }
  const NOTIFICATION_TOGGLE_NOTIFICATIONS: Record<
    'alreadyDisabled' | 'channelRequired',
    TranslationKey
  > = {
    alreadyDisabled: 'settings.notificationEventToggle.notifications.alreadyDisabled',
    channelRequired: 'settings.notificationEventToggle.notifications.channelRequired',
  }
  const NOTIFICATION_TOGGLE_ERROR_KEY: TranslationKey =
    'settings.notificationEventToggle.errors.update'

  $effect(() => {
    if (availableChannels.length === 0) {
      selectedChannelId = ''
      return
    }
    const hasSelection = availableChannels.some((channel) => channel.id === selectedChannelId)
    if (!hasSelection) {
      selectedChannelId = availableChannels[0]?.id ?? ''
    }
  })

  function selectedChannel(): NotificationChannel | undefined {
    return availableChannels.find((channel) => channel.id === selectedChannelId)
  }

  async function handleToggle(eventType: NotificationRuleEventType, nextEnabled: boolean) {
    const channel = selectedChannel()
    if (!channel) {
      toastStore.error(i18nStore.t(NOTIFICATION_TOGGLE_NOTIFICATIONS.channelRequired))
      return
    }

    const currentRule = findNotificationToggleRule(rules, channel.id, eventType.event_type)
    actionKey = `${channel.id}:${eventType.event_type}`

    try {
      if (currentRule) {
        if (nextEnabled) {
          await onUpdate(currentRule.id, { is_enabled: true })
        } else {
          await onDelete(currentRule.id)
        }
      } else if (nextEnabled) {
        await onCreate(buildNotificationToggleRuleInput(eventType, channel))
      } else {
        toastStore.info(
          i18nStore.t(NOTIFICATION_TOGGLE_NOTIFICATIONS.alreadyDisabled, {
            event: eventType.label,
            channel: channel.name,
          }),
        )
        return
      }

      toastStore.success(
        i18nStore.t(NOTIFICATION_TOGGLE_MESSAGE_KEYS[nextEnabled ? 'enabled' : 'disabled'], {
          event: eventType.label,
          channel: channel.name,
        }),
      )
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, i18nStore.t(NOTIFICATION_TOGGLE_ERROR_KEY)))
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="border-border bg-card rounded-md border">
  <div class="border-border flex flex-wrap items-start justify-between gap-4 border-b px-5 py-4">
    <div>
      <h3 class="text-foreground text-base font-semibold">
        {i18nStore.t('settings.notificationEventToggle.heading')}
      </h3>
      <p class="text-muted-foreground mt-1 text-sm">
        {i18nStore.t('settings.notificationEventToggle.description')}
      </p>
    </div>

    <div class="w-full max-w-xs space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.14em] uppercase">
        {i18nStore.t('settings.notificationEventToggle.labels.activeChannel')}
      </span>
      <Select.Root
        type="single"
        value={selectedChannelId}
        onValueChange={(value) => (selectedChannelId = value || '')}
      >
        <Select.Trigger class="w-full">
          {selectedChannel()?.name ??
            i18nStore.t('settings.notificationEventToggle.placeholders.selectChannel')}
        </Select.Trigger>
        <Select.Content>
          {#each availableChannels as channel (channel.id)}
            <Select.Item value={channel.id}>{channel.name}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  {#if availableChannels.length === 0}
    <div class="px-5 py-5">
      <div class="border-border bg-muted/40 rounded-md border px-4 py-3 text-sm">
        {i18nStore.t('settings.notificationEventToggle.messages.enableChannelFirst')}
      </div>
    </div>
  {:else}
    <div class="space-y-5 px-5 py-5">
      {#each groupedEvents as group (group.group)}
        <section class="space-y-3">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h4 class="text-foreground text-sm font-semibold">{group.group}</h4>
              <p class="text-muted-foreground mt-1 text-xs">
                {i18nStore.t('settings.notificationEventToggle.labels.deliveryStatus', {
                  channel:
                    selectedChannel()?.name ??
                    i18nStore.t('settings.notificationEventToggle.placeholders.selectedChannel'),
                })}
              </p>
            </div>
            <Badge variant="outline">
              {i18nStore.t('settings.notificationEventToggle.badges.eventCount', {
                count: group.events.length,
              })}
            </Badge>
          </div>

          <div class="space-y-2">
            {#each group.events as eventType (eventType.event_type)}
              {@const currentRule = findNotificationToggleRule(
                rules,
                selectedChannelId,
                eventType.event_type,
              )}
              {@const busy = actionKey === `${selectedChannelId}:${eventType.event_type}`}
              <div
                class="border-border/70 bg-background/80 flex items-center gap-3 rounded-md border px-4 py-3"
              >
                <Switch
                  checked={currentRule?.is_enabled ?? false}
                  disabled={busy}
                  aria-label={i18nStore.t(
                    currentRule?.is_enabled
                      ? 'settings.notificationEventToggle.aria.disable'
                      : 'settings.notificationEventToggle.aria.enable',
                    { event: eventType.label },
                  )}
                  onCheckedChange={(checked) => void handleToggle(eventType, Boolean(checked))}
                />

                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="text-foreground text-sm font-medium">{eventType.label}</span>
                    <Badge variant="secondary" class="capitalize">{eventType.level}</Badge>
                  </div>
                  <p class="text-muted-foreground mt-1 font-mono text-[11px]">
                    {eventType.event_type}
                  </p>
                </div>

                <div class="text-right text-xs">
                  <div class="text-foreground font-medium">
                    {busy
                      ? i18nStore.t('settings.notificationEventToggle.status.saving')
                      : currentRule?.is_enabled
                        ? i18nStore.t('settings.notificationEventToggle.states.enabled')
                        : i18nStore.t('settings.notificationEventToggle.states.disabled')}
                  </div>
                  <div class="text-muted-foreground mt-1">
                    {currentRule
                      ? i18nStore.t('settings.notificationEventToggle.hints.usesDefaultMessage')
                      : i18nStore.t(
                          'settings.notificationEventToggle.hints.autoCreatesDefaultRule',
                        )}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</div>
