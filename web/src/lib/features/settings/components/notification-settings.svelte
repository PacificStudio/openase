<script lang="ts">
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { ApiError } from '$lib/api/client'
  import {
    createNotificationChannel,
    createNotificationRule,
    deleteNotificationChannel,
    deleteNotificationRule,
    listNotificationChannels,
    listNotificationEventTypes,
    listNotificationRules,
    testNotificationChannel,
    updateNotificationChannel,
    updateNotificationRule,
  } from '$lib/api/openase'
  import type {
    NotificationChannel,
    NotificationRule,
    NotificationRuleEventType,
  } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { Separator } from '$ui/separator'
  import type { ChannelCreateInput, ChannelUpdateInput } from '../notification-channels'
  import type { RuleCreateInput, RuleUpdateInput } from '../notification-rules'
  import NotificationChannelPanel from './notification-channel-panel.svelte'
  import NotificationRulePanel from './notification-rule-panel.svelte'

  const notificationsCapability = capabilityCatalog.notificationsSettings

  let channels = $state<NotificationChannel[]>([])
  let rules = $state<NotificationRule[]>([])
  let eventTypes = $state<NotificationRuleEventType[]>([])
  let loading = $state(false)
  let error = $state('')

  $effect(() => {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (!orgId || !projectId) {
      channels = []
      rules = []
      eventTypes = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [eventTypePayload, channelPayload, rulePayload] = await Promise.all([
          listNotificationEventTypes(),
          listNotificationChannels(orgId),
          listNotificationRules(projectId),
        ])
        if (cancelled) return
        eventTypes = eventTypePayload.event_types
        channels = channelPayload.channels
        rules = rulePayload.rules
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load notification settings.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  async function refreshChannels() {
    const orgId = appStore.currentOrg?.id
    if (!orgId) {
      throw new Error('No organization selected.')
    }
    const payload = await listNotificationChannels(orgId)
    channels = payload.channels
  }

  async function refreshRules() {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      throw new Error('No project selected.')
    }
    const payload = await listNotificationRules(projectId)
    rules = payload.rules
  }

  function unwrapApiError(caughtError: unknown, fallback: string): never {
    if (caughtError instanceof ApiError) {
      throw new Error(caughtError.detail)
    }
    if (caughtError instanceof Error && caughtError.message) {
      throw caughtError
    }
    throw new Error(fallback)
  }

  async function handleCreateChannel(input: ChannelCreateInput): Promise<NotificationChannel> {
    try {
      const orgId = appStore.currentOrg?.id
      if (!orgId) throw new Error('No organization selected.')
      const payload = await createNotificationChannel(orgId, input)
      await Promise.all([refreshChannels(), refreshRules()])
      return payload.channel
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to create channel.')
    }
  }

  async function handleUpdateChannel(
    channelId: string,
    input: ChannelUpdateInput,
  ): Promise<NotificationChannel> {
    try {
      const payload = await updateNotificationChannel(channelId, input)
      await Promise.all([refreshChannels(), refreshRules()])
      return payload.channel
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to update channel.')
    }
  }

  async function handleDeleteChannel(channelId: string): Promise<void> {
    try {
      await deleteNotificationChannel(channelId)
      await Promise.all([refreshChannels(), refreshRules()])
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to delete channel.')
    }
  }

  async function handleToggleChannel(
    channelId: string,
    isEnabled: boolean,
  ): Promise<NotificationChannel> {
    try {
      const payload = await updateNotificationChannel(channelId, { is_enabled: isEnabled })
      await Promise.all([refreshChannels(), refreshRules()])
      return payload.channel
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to update channel state.')
    }
  }

  async function handleTestChannel(channelId: string): Promise<void> {
    try {
      await testNotificationChannel(channelId)
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to send channel test.')
    }
  }

  async function handleCreateRule(input: RuleCreateInput): Promise<NotificationRule> {
    try {
      const projectId = appStore.currentProject?.id
      if (!projectId) throw new Error('No project selected.')
      const payload = await createNotificationRule(projectId, input)
      await refreshRules()
      return payload.rule
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to create rule.')
    }
  }

  async function handleUpdateRule(
    ruleId: string,
    input: RuleUpdateInput,
  ): Promise<NotificationRule> {
    try {
      const payload = await updateNotificationRule(ruleId, input)
      await refreshRules()
      return payload.rule
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to update rule.')
    }
  }

  async function handleDeleteRule(ruleId: string): Promise<void> {
    try {
      await deleteNotificationRule(ruleId)
      await refreshRules()
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to delete rule.')
    }
  }

  async function handleToggleRule(
    ruleId: string,
    isEnabled: boolean,
  ): Promise<NotificationRule> {
    try {
      const payload = await updateNotificationRule(ruleId, { is_enabled: isEnabled })
      await refreshRules()
      return payload.rule
    } catch (caughtError) {
      unwrapApiError(caughtError, 'Failed to update rule state.')
    }
  }
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Notifications</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(notificationsCapability.state)}`}
      >
        {capabilityStateLabel(notificationsCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{notificationsCapability.summary}</p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading notification settings…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else}
    <div class="space-y-6">
      <NotificationChannelPanel
        {channels}
        onCreate={handleCreateChannel}
        onUpdate={handleUpdateChannel}
        onDelete={handleDeleteChannel}
        onToggle={handleToggleChannel}
        onTest={handleTestChannel}
      />
      <NotificationRulePanel
        {channels}
        {eventTypes}
        {rules}
        onCreate={handleCreateRule}
        onUpdate={handleUpdateRule}
        onDelete={handleDeleteRule}
        onToggle={handleToggleRule}
      />
    </div>
  {/if}
</div>
