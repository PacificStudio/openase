<script lang="ts">
  import { ArrowLeft, LoaderCircle } from '@lucide/svelte'
  import type { createTicketDetailStore } from '../store.svelte'
  import EventTimelinePanel from './EventTimelinePanel.svelte'
  import RepoStatusPanel from './RepoStatusPanel.svelte'
  import StreamStatusPills from './StreamStatusPills.svelte'
  import TicketGraphPanel from './TicketGraphPanel.svelte'
  import TicketSummaryCard from './TicketSummaryCard.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createTicketDetailStore>
  } = $props()
</script>

<svelte:head>
  <title>
    {controller.detail ? `${controller.detail.ticket.identifier} · Ticket Detail` : 'Ticket Detail'}
    · OpenASE
  </title>
</svelte:head>

<div
  class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.14),_transparent_34%),radial-gradient(circle_at_top_right,_rgba(14,165,233,0.12),_transparent_30%),linear-gradient(180deg,_rgba(248,250,252,0.98),_rgba(241,245,249,0.96))]"
>
  <div class="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <a
        href="/"
        class="border-border/70 bg-background/85 text-foreground hover:border-foreground/20 inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition"
      >
        <ArrowLeft class="size-4" />
        Back to board
      </a>

      <StreamStatusPills
        ticketStreamState={controller.ticketStreamState}
        activityStreamState={controller.activityStreamState}
        hookStreamState={controller.hookStreamState}
      />
    </div>

    {#if controller.errorMessage}
      <div
        class="border-destructive/25 bg-destructive/10 text-destructive rounded-3xl border px-4 py-3 text-sm"
      >
        {controller.errorMessage}
      </div>
    {/if}

    {#if controller.loading && !controller.detail}
      <div
        class="border-border/70 bg-background/80 flex min-h-[24rem] items-center justify-center rounded-[2rem] border"
      >
        <div class="text-muted-foreground flex items-center gap-3 text-sm">
          <LoaderCircle class="size-4 animate-spin" />
          <span>Loading ticket detail…</span>
        </div>
      </div>
    {:else if !controller.projectId || !controller.ticketId}
      <div
        class="border-border/80 bg-background/80 text-muted-foreground rounded-[2rem] border border-dashed px-6 py-8 text-sm"
      >
        Ticket detail URLs require both `project` and `id` query parameters.
      </div>
    {:else if controller.detail}
      <div class="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_24rem]">
        <div class="space-y-6">
          <TicketSummaryCard
            detail={controller.detail}
            project={controller.project}
            projectId={controller.projectId}
            refreshing={controller.refreshing}
          />

          <RepoStatusPanel repoScopes={controller.detail.repo_scopes} />

          <EventTimelinePanel
            title="Execution stream"
            description="Newest execution events land first, and SSE appends new lines automatically."
            events={controller.detail.activity}
            emptyMessage="No execution events have been recorded yet."
          />
        </div>

        <div class="space-y-6">
          <EventTimelinePanel
            title="Hook history"
            description="Hook-tagged events are broken out so failures are visible without scanning the full log."
            variant="hooks"
            events={controller.detail.hook_history}
            emptyMessage="No hook events have been recorded for this ticket yet."
          />

          <TicketGraphPanel detail={controller.detail} />
        </div>
      </div>
    {/if}
  </div>
</div>
