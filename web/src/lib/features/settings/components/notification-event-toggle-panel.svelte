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
      toastStore.error('Enable a notification channel before configuring event toggles.')
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
        toastStore.info(`${eventType.label} is already disabled for ${channel.name}.`)
        return
      }

      toastStore.success(
        nextEnabled
          ? `${eventType.label} enabled for ${channel.name}.`
          : `${eventType.label} disabled for ${channel.name}.`,
      )
    } catch (caughtError) {
      toastStore.error(actionErrorMessage(caughtError, 'Failed to update notification toggle.'))
    } finally {
      actionKey = ''
    }
  }
</script>

<div class="border-border bg-card rounded-2xl border">
  <div class="border-border flex flex-wrap items-start justify-between gap-4 border-b px-5 py-4">
    <div>
      <h3 class="text-foreground text-base font-semibold">Event Toggles</h3>
      <p class="text-muted-foreground mt-1 text-sm">
        Pick one enabled channel, then turn supported notification events on or off by group.
      </p>
    </div>

    <div class="w-full max-w-xs space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.14em] uppercase">
        Active channel
      </span>
      <Select.Root
        type="single"
        value={selectedChannelId}
        onValueChange={(value) => (selectedChannelId = value || '')}
      >
        <Select.Trigger class="w-full">
          {selectedChannel()?.name ?? 'Select enabled channel'}
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
      <div class="border-border bg-muted/40 rounded-xl border px-4 py-3 text-sm">
        Create and enable at least one notification channel before configuring grouped event
        toggles.
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
                Delivery status for {selectedChannel()?.name ?? 'the selected channel'}.
              </p>
            </div>
            <Badge variant="outline">{group.events.length} events</Badge>
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
                class="border-border/70 bg-background/80 flex items-center gap-3 rounded-xl border px-4 py-3"
              >
                <Switch
                  checked={currentRule?.is_enabled ?? false}
                  disabled={busy}
                  aria-label={`${currentRule?.is_enabled ? 'Disable' : 'Enable'} ${eventType.label}`}
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
                    {busy ? 'Saving…' : currentRule?.is_enabled ? 'Enabled' : 'Disabled'}
                  </div>
                  <div class="text-muted-foreground mt-1">
                    {currentRule ? 'Uses default message' : 'Creates default rule on first enable'}
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
