<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Switch } from '$ui/switch'
  import { Pencil, Play, Trash2, Plus, CalendarClock } from '@lucide/svelte'

  let {
    jobs,
    actionJobId = null,
    onNewJob,
    onEditJob,
    onToggleEnabled,
    onTriggerJob,
    onDeleteJob,
  }: {
    jobs: ScheduledJob[]
    actionJobId?: string | null
    onNewJob?: () => void
    onEditJob?: (job: ScheduledJob) => void
    onToggleEnabled?: (job: ScheduledJob) => void
    onTriggerJob?: (job: ScheduledJob) => void
    onDeleteJob?: (job: ScheduledJob) => void
  } = $props()
</script>

{#if jobs.length === 0}
  <div
    class="border-border bg-card animate-fade-in-up rounded-xl border border-dashed px-4 py-14 text-center"
  >
    <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
      <CalendarClock class="text-muted-foreground size-5" />
    </div>
    <p class="text-foreground text-sm font-medium">No scheduled jobs</p>
    <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-sm">
      Scheduled jobs create tickets automatically on a cron schedule — useful for recurring tasks
      like daily reports or periodic syncs.
    </p>
    <Button variant="outline" size="sm" class="mt-4" onclick={() => onNewJob?.()}>
      <Plus class="mr-1.5 size-3.5" />
      Create job
    </Button>
  </div>
{:else}
  <div class="space-y-2">
    {#each jobs as job (job.id)}
      {@const busy = actionJobId === job.id}
      <div class="border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3">
        <!-- Toggle -->
        <Switch
          checked={job.is_enabled}
          disabled={busy}
          onCheckedChange={() => onToggleEnabled?.(job)}
          aria-label="{job.is_enabled ? 'Disable' : 'Enable'} {job.name}"
          class="shrink-0"
        />

        <!-- Main info -->
        <button type="button" class="min-w-0 flex-1 text-left" onclick={() => onEditJob?.(job)}>
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-foreground text-sm font-semibold hover:underline">
              {job.name}
            </span>
            <span class="text-muted-foreground font-mono text-xs">
              {job.cron_expression}
            </span>
          </div>
          <div
            class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs"
          >
            <span>Status {job.ticket_template.status || 'Unassigned'}</span>
            <span>
              Next {job.next_run_at ? formatRelativeTime(job.next_run_at) : '—'}
            </span>
            {#if job.last_run_at}
              <span>
                Last {formatRelativeTime(job.last_run_at)}
              </span>
            {/if}
          </div>
        </button>

        <!-- Actions -->
        <div class="flex shrink-0 items-center gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Edit job"
            title="Edit job"
            onclick={() => onEditJob?.(job)}
          >
            <Pencil class="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Run once"
            title="Run once"
            disabled={busy}
            onclick={() => onTriggerJob?.(job)}
          >
            <Play class="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Delete job"
            title="Delete job"
            disabled={busy}
            class="text-muted-foreground hover:text-destructive"
            onclick={() => onDeleteJob?.(job)}
          >
            <Trash2 class="size-3.5" />
          </Button>
        </div>
      </div>
    {/each}
  </div>
{/if}
