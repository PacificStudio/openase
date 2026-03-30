<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Plus } from '@lucide/svelte'

  let {
    jobs,
    selectedJobId = '',
    workflowLabelById,
    onSelect,
    onNewJob,
  }: {
    jobs: ScheduledJob[]
    selectedJobId?: string
    workflowLabelById: Map<string, string>
    onSelect?: (job: ScheduledJob) => void
    onNewJob?: () => void
  } = $props()
</script>

<div class="border-border flex w-52 shrink-0 flex-col border-r">
  <div class="border-border flex items-center justify-between border-b px-3 py-2.5">
    <span class="text-muted-foreground text-xs font-medium">{jobs.length} jobs</span>
    <button
      type="button"
      class="text-muted-foreground hover:bg-muted hover:text-foreground rounded p-1 transition-colors"
      title="Add job"
      aria-label="Add job"
      onclick={() => onNewJob?.()}
    >
      <Plus class="size-3.5" />
    </button>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#each jobs as job (job.id)}
      <button
        type="button"
        class="hover:bg-muted/40 border-border flex w-full flex-col gap-1 border-b px-3 py-2.5 text-left transition-colors {selectedJobId ===
        job.id
          ? 'bg-muted/50'
          : ''}"
        onclick={() => onSelect?.(job)}
      >
        <div class="flex w-full items-center justify-between gap-2">
          <span class="text-foreground min-w-0 truncate text-xs font-medium">{job.name}</span>
          <span
            class="shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium {job.is_enabled
              ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
              : 'bg-slate-500/10 text-slate-700 dark:text-slate-300'}"
          >
            {job.is_enabled ? 'On' : 'Off'}
          </span>
        </div>
        <div class="text-muted-foreground truncate font-mono text-[11px]">
          {job.cron_expression}
        </div>
        <div class="text-muted-foreground truncate text-[11px]">
          {workflowLabelById.get(job.workflow_id) ?? 'Unknown'} · Next {job.next_run_at
            ? formatRelativeTime(job.next_run_at)
            : '—'}
        </div>
      </button>
    {:else}
      <div class="text-muted-foreground px-3 py-6 text-center text-xs">No jobs yet.</div>
    {/each}
  </div>
</div>
