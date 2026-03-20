<script lang="ts">
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Search } from '@lucide/svelte'
  import type { ActivityEntry } from '../types'
  import ActivityTimeline from './activity-timeline.svelte'

  const now = new Date()
  const h = (hours: number) => new Date(now.getTime() - hours * 3_600_000).toISOString()

  const mockEntries: ActivityEntry[] = [
    {
      id: '1',
      eventType: 'ticket_created',
      message: 'Ticket created: Add dark mode toggle',
      timestamp: h(0.5),
      ticketIdentifier: 'OAS-142',
    },
    {
      id: '2',
      eventType: 'agent_started',
      message: 'Agent began working on OAS-142',
      timestamp: h(0.8),
      ticketIdentifier: 'OAS-142',
      agentName: 'claude-alpha',
    },
    {
      id: '3',
      eventType: 'pr_opened',
      message: 'Pull request #87 opened for OAS-140',
      timestamp: h(1.2),
      ticketIdentifier: 'OAS-140',
      agentName: 'claude-beta',
    },
    {
      id: '4',
      eventType: 'comment_added',
      message: 'Review comment added on PR #87',
      timestamp: h(2),
      ticketIdentifier: 'OAS-140',
    },
    {
      id: '5',
      eventType: 'agent_completed',
      message: 'Agent completed work on OAS-139',
      timestamp: h(3),
      ticketIdentifier: 'OAS-139',
      agentName: 'claude-alpha',
    },
    {
      id: '6',
      eventType: 'status_changed',
      message: 'OAS-138 moved to In Review',
      timestamp: h(4.5),
      ticketIdentifier: 'OAS-138',
    },
    {
      id: '7',
      eventType: 'hook_failed',
      message: 'Pre-commit hook failed on OAS-137',
      timestamp: h(5),
      ticketIdentifier: 'OAS-137',
      agentName: 'claude-gamma',
    },
    {
      id: '8',
      eventType: 'pr_merged',
      message: 'Pull request #85 merged for OAS-136',
      timestamp: h(8),
      ticketIdentifier: 'OAS-136',
    },
    {
      id: '9',
      eventType: 'ticket_created',
      message: 'Ticket created: Refactor auth middleware',
      timestamp: h(25),
      ticketIdentifier: 'OAS-141',
    },
    {
      id: '10',
      eventType: 'agent_started',
      message: 'Agent began working on OAS-135',
      timestamp: h(26),
      ticketIdentifier: 'OAS-135',
      agentName: 'claude-beta',
    },
    { id: '11', eventType: 'budget_alert', message: 'Daily budget 80% consumed', timestamp: h(27) },
    {
      id: '12',
      eventType: 'agent_completed',
      message: 'Agent completed work on OAS-134',
      timestamp: h(28),
      ticketIdentifier: 'OAS-134',
      agentName: 'claude-alpha',
    },
    {
      id: '13',
      eventType: 'status_changed',
      message: 'OAS-133 moved to Done',
      timestamp: h(30),
      ticketIdentifier: 'OAS-133',
    },
    {
      id: '14',
      eventType: 'pr_opened',
      message: 'Pull request #84 opened for OAS-132',
      timestamp: h(32),
      ticketIdentifier: 'OAS-132',
      agentName: 'claude-gamma',
    },
    {
      id: '15',
      eventType: 'agent_stalled',
      message: 'Agent stalled on OAS-131 — no heartbeat',
      timestamp: h(50),
      ticketIdentifier: 'OAS-131',
      agentName: 'claude-beta',
    },
    {
      id: '16',
      eventType: 'ticket_created',
      message: 'Ticket created: Fix pagination bug',
      timestamp: h(51),
      ticketIdentifier: 'OAS-130',
    },
    {
      id: '17',
      eventType: 'hook_failed',
      message: 'CI pipeline failed on PR #82',
      timestamp: h(52),
      ticketIdentifier: 'OAS-129',
      agentName: 'claude-alpha',
    },
    {
      id: '18',
      eventType: 'pr_merged',
      message: 'Pull request #81 merged for OAS-128',
      timestamp: h(54),
      ticketIdentifier: 'OAS-128',
    },
  ]

  const eventTypes = [
    { value: 'all', label: 'All events' },
    { value: 'ticket_created', label: 'Ticket created' },
    { value: 'agent_started', label: 'Agent started' },
    { value: 'agent_completed', label: 'Agent completed' },
    { value: 'hook_failed', label: 'Hook failed' },
    { value: 'pr_opened', label: 'PR opened' },
    { value: 'pr_merged', label: 'PR merged' },
    { value: 'status_changed', label: 'Status changed' },
  ]

  let searchQuery = $state('')
  let selectedType = $state<string>('all')

  const filtered = $derived(
    mockEntries.filter((e) => {
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
</script>

<div class="mx-auto w-full max-w-3xl space-y-6">
  <div>
    <h1 class="text-foreground text-lg font-semibold">Activity</h1>
    <p class="text-muted-foreground mt-1 text-sm">
      Event log across all tickets, agents, and integrations.
    </p>
  </div>

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
        {eventTypes.find((t) => t.value === selectedType)?.label ?? 'All events'}
      </Select.Trigger>
      <Select.Content>
        {#each eventTypes as t (t.value)}
          <Select.Item value={t.value}>{t.label}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>

  {#if filtered.length > 0}
    <ActivityTimeline entries={filtered} />
  {:else}
    <div class="text-muted-foreground py-16 text-center text-sm">No events match your filters.</div>
  {/if}
</div>
