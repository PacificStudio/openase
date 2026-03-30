<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import type { AgentOutputEntry, AgentStepEntry } from '$lib/api/contracts'
  import type { StreamConnectionState } from '$lib/api/sse'
  import { formatRelativeTime } from '$lib/utils'
  import type { AgentInstance } from '../types'

  let {
    open = $bindable(false),
    agent,
    entries,
    steps,
    loading = false,
    error = '',
    streamState = 'idle',
    onOpenChange,
  }: {
    open?: boolean
    agent: AgentInstance | null
    entries: AgentOutputEntry[]
    steps: AgentStepEntry[]
    loading?: boolean
    error?: string
    streamState?: StreamConnectionState
    onOpenChange?: (open: boolean) => void
  } = $props()

  const streamStateLabel: Record<StreamConnectionState, string> = {
    idle: 'Idle',
    connecting: 'Connecting',
    live: 'Live',
    retrying: 'Retrying',
  }
  let lastReportedOpen = $state(open)

  $effect(() => {
    if (open === lastReportedOpen) {
      return
    }

    lastReportedOpen = open
    onOpenChange?.(open)
  })

  function formatOutputTimestamp(value: string) {
    return new Date(value).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-2xl">
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <div class="flex flex-wrap items-center gap-2">
        <SheetTitle>{agent?.name ?? 'Agent output'}</SheetTitle>
        <Badge variant={streamState === 'live' ? 'default' : 'secondary'}>
          {streamStateLabel[streamState]}
        </Badge>
        {#if agent?.runtimePhase}
          <Badge variant="outline">{agent.runtimePhase}</Badge>
        {/if}
        {#if agent?.currentStepStatus}
          <Badge variant="outline">{agent.currentStepStatus}</Badge>
        {/if}
      </div>
      <SheetDescription>
        Dedicated runtime trace and human-readable steps for the selected agent. Snapshot history
        loads first, then live trace and step feeds append independently as new runtime events
        arrive.
      </SheetDescription>
      {#if agent}
        <div class="text-muted-foreground mt-3 flex flex-wrap gap-4 text-xs">
          <span>{agent.providerName}</span>
          {#if agent.sessionId}
            <span>session {agent.sessionId}</span>
          {/if}
          {#if agent.currentTicket}
            <span>{agent.currentTicket.identifier}</span>
          {/if}
          {#if agent.currentStepSummary}
            <span>step {agent.currentStepSummary}</span>
          {/if}
          {#if agent.lastHeartbeat}
            <span>heartbeat {formatRelativeTime(agent.lastHeartbeat)}</span>
          {/if}
        </div>
      {/if}
    </SheetHeader>

    <div class="flex min-h-0 flex-1 flex-col bg-slate-950 text-slate-100">
      {#if error}
        <div
          class="m-6 rounded-md border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200"
        >
          {error}
        </div>
      {:else if loading}
        <div class="px-6 py-6 text-sm text-slate-300">Loading agent runtime…</div>
      {:else}
        <div class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
          <div class="space-y-6">
            <section class="rounded-xl border border-white/10 bg-white/5 p-4">
              <div class="mb-3 flex items-center justify-between gap-2">
                <h3 class="text-sm font-semibold text-slate-100">Action Timeline</h3>
                {#if agent?.currentStepChangedAt}
                  <span class="text-xs text-slate-400">
                    Updated {formatRelativeTime(agent.currentStepChangedAt)}
                  </span>
                {/if}
              </div>
              {#if steps.length === 0}
                <div class="text-sm text-slate-400">
                  No human-readable steps have been projected yet.
                </div>
              {:else}
                <div class="space-y-3">
                  {#each steps as step (step.id)}
                    <div class="rounded-lg border border-white/10 bg-slate-950/40 px-3 py-3">
                      <div
                        class="flex flex-wrap items-center gap-2 text-xs tracking-[0.16em] text-slate-400 uppercase"
                      >
                        <span>{step.step_status}</span>
                        <span>{formatOutputTimestamp(step.created_at)}</span>
                      </div>
                      <div class="mt-2 text-sm text-slate-100">
                        {step.summary || 'No summary recorded.'}
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </section>

            <section class="space-y-4">
              <div class="flex items-center justify-between gap-2">
                <h3 class="text-sm font-semibold text-slate-100">Raw Trace</h3>
                <span class="text-xs text-slate-400">{entries.length} entries</span>
              </div>
              {#if entries.length === 0}
                <div
                  class="rounded-xl border border-white/10 bg-white/5 px-4 py-4 text-sm text-slate-400"
                >
                  No trace events have been recorded for this runtime yet.
                </div>
              {:else}
                <div class="space-y-4">
                  {#each entries as entry (entry.id)}
                    <section class="rounded-lg border border-white/10 bg-white/5">
                      <div
                        class="flex flex-wrap items-center gap-2 border-b border-white/10 px-4 py-2 text-[11px] tracking-[0.18em] text-slate-400 uppercase"
                      >
                        <span>{formatOutputTimestamp(entry.created_at)}</span>
                        <span class="rounded-full border border-white/10 px-2 py-0.5"
                          >{entry.stream}</span
                        >
                        {#if entry.ticket_id}
                          <span class="rounded-full border border-white/10 px-2 py-0.5">
                            ticket linked
                          </span>
                        {/if}
                      </div>
                      <pre
                        class="overflow-x-auto px-4 py-3 font-mono text-xs leading-6 whitespace-pre-wrap">{entry.output}</pre>
                    </section>
                  {/each}
                </div>
              {/if}
            </section>
          </div>
        </div>
      {/if}

      {#if agent?.lastError}
        <div class="border-t border-red-500/20 bg-red-500/10 px-6 py-3 text-xs text-red-200">
          Last runtime error: {agent.lastError}
        </div>
      {/if}
    </div>
  </SheetContent>
</Sheet>
