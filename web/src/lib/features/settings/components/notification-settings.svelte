<script lang="ts">
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

  async function handleToggleRule(ruleId: string, isEnabled: boolean): Promise<NotificationRule> {
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
    <h2 class="text-foreground text-base font-semibold">Notifications</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage notification channels, rules, and delivery controls.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="space-y-6">
      <!-- Skeleton: channels -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="bg-muted h-4 w-36 animate-pulse rounded"></div>
          <div class="bg-muted h-8 w-24 animate-pulse rounded-md"></div>
        </div>
        {#each { length: 2 } as _}
          <div class="border-border bg-card flex items-center gap-3 rounded-lg border px-4 py-3">
            <div class="bg-muted h-5 w-9 shrink-0 animate-pulse rounded-full"></div>
            <div class="flex-1 space-y-1.5">
              <div class="bg-muted h-4 w-28 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
            </div>
            <div class="flex items-center gap-1">
              <div class="bg-muted size-7 animate-pulse rounded"></div>
              <div class="bg-muted size-7 animate-pulse rounded"></div>
            </div>
          </div>
        {/each}
      </div>
      <!-- Skeleton: rules -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
          <div class="bg-muted h-8 w-20 animate-pulse rounded-md"></div>
        </div>
        {#each { length: 2 } as _}
          <div class="border-border bg-card flex items-center gap-3 rounded-lg border px-4 py-3">
            <div class="bg-muted h-5 w-9 shrink-0 animate-pulse rounded-full"></div>
            <div class="flex-1 space-y-1.5">
              <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-44 animate-pulse rounded"></div>
            </div>
            <div class="bg-muted size-7 animate-pulse rounded"></div>
          </div>
        {/each}
      </div>
    </div>
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
