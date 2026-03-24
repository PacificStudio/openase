<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'

  let {
    jobs,
    selectedJobId = '',
    workflowLabelById,
    onSelect,
  }: {
    jobs: ScheduledJob[]
    selectedJobId?: string
    workflowLabelById: Map<string, string>
    onSelect?: (job: ScheduledJob) => void
  } = $props()
</script>

<div class="border-border w-80 shrink-0 border-r">
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <span class="text-foreground text-sm font-medium">Configured jobs</span>
    <span class="text-muted-foreground text-xs">{jobs.length} total</span>
  </div>

  <div class="divide-border divide-y">
    {#each jobs as job (job.id)}
      <button
        type="button"
        class={`hover:bg-muted/40 flex w-full flex-col items-start gap-2 px-4 py-3 text-left transition-colors ${
          selectedJobId === job.id ? 'bg-muted/50' : ''
        }`}
        onclick={() => onSelect?.(job)}
      >
        <div class="flex w-full items-center justify-between gap-3">
          <span class="text-foreground text-sm font-medium">{job.name}</span>
          <span
            class={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
              job.is_enabled
                ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
                : 'bg-slate-500/10 text-slate-700 dark:text-slate-300'
            }`}
          >
            {job.is_enabled ? 'Enabled' : 'Disabled'}
          </span>
        </div>
        <div class="text-muted-foreground text-xs">{job.cron_expression}</div>
        <div class="text-muted-foreground flex flex-wrap gap-2 text-[11px]">
          <span>{workflowLabelById.get(job.workflow_id) ?? 'Unknown workflow'}</span>
          <span>Next {job.next_run_at ? formatRelativeTime(job.next_run_at) : 'not scheduled'}</span
          >
        </div>
      </button>
    {:else}
      <div class="px-4 py-8 text-center text-xs text-muted-foreground">
        No scheduled jobs yet. Create the first recurring run from the editor.
      </div>
    {/each}
  </div>
</div>
