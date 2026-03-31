<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { listActivity, listTickets } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Search } from '@lucide/svelte'
  import type { ActivityEntry } from '../types'
  import { activityEventFilterOptions } from '../event-catalog'
  import ActivityTimeline from './activity-timeline.svelte'

  let entries = $state<ActivityEntry[]>([])
  let loading = $state(false)
  let error = $state('')
  let searchQuery = $state('')
  let selectedType = $state<string>('all')

  const filtered = $derived(
    entries.filter((e) => {
      if (selectedType !== 'all' && e.eventType !== selectedType) return false
      if (searchQuery) {
        const q = searchQuery.toLowerCase()
        return (
          e.message.toLowerCase().includes(q) ||
          e.ticketIdentifier?.toLowerCase().includes(q) ||
          e.agentName?.toLowerCase().includes(q)
        )
      }
      return true
    }),
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      entries = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [activityPayload, ticketPayload] = await Promise.all([
          listActivity(projectId, { limit: 100 }),
          listTickets(projectId),
        ])
        if (cancelled) return

        const ticketIdentifiers = new Map(
          ticketPayload.tickets.map((ticket) => [ticket.id, ticket.identifier]),
        )

        entries = activityPayload.events.map((event) => ({
          id: event.id,
          eventType: normalizeEventType(event.event_type),
          message: event.message,
          timestamp: event.created_at,
          ticketIdentifier: event.ticket_id
            ? (ticketIdentifiers.get(event.ticket_id) ?? event.ticket_id)
            : undefined,
          agentName: agentNameFromMetadata(event.metadata),
        }))
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load activity.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/activity/stream`, {
      onEvent: () => {
        void load()
      },
      onError: (streamError) => {
        console.error('Activity stream error:', streamError)
      },
    })

    return () => {
      cancelled = true
      disconnect()
    }
  })

  function agentNameFromMetadata(metadata: Record<string, unknown>) {
    const value = metadata.agent_name
    return typeof value === 'string' ? value : undefined
  }

  function normalizeEventType(eventType: string) {
    switch (eventType) {
      case 'ticket_created':
        return 'ticket.created'
      case 'hook_failed':
        return 'hook.failed'
      case 'pr_opened':
        return 'pr.opened'
      case 'pr_merged':
        return 'pr.merged'
      case 'status_changed':
        return 'ticket.status_changed'
      default:
        return eventType
    }
  }
</script>

<PageScaffold title="Activity" description="Runtime events and agent lifecycle updates.">
  <div class="w-full max-w-4xl space-y-4">
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative min-w-48 flex-1">
        <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input placeholder="Search events..." class="pl-9" bind:value={searchQuery} />
      </div>
      <Select.Root
        type="single"
        onValueChange={(v) => {
          selectedType = v || 'all'
        }}
      >
        <Select.Trigger class="w-44">
          {activityEventFilterOptions.find((t) => t.value === selectedType)?.label ?? 'All events'}
        </Select.Trigger>
        <Select.Content>
          {#each activityEventFilterOptions as t (t.value)}
            <Select.Item value={t.value}>{t.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    {#if loading}
      <div class="text-muted-foreground py-16 text-sm">Loading activity…</div>
    {:else if error}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {:else if filtered.length > 0}
      <ActivityTimeline entries={filtered} />
    {:else}
      <div class="text-muted-foreground py-16 text-sm">No events match your filters.</div>
    {/if}
  </div>
</PageScaffold>
