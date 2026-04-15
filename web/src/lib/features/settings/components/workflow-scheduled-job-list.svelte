<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Switch } from '$ui/switch'
  import { Pencil, Play, Trash2, Plus, CalendarClock } from '@lucide/svelte'
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'

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

  const toggleActionKeys: Record<'enable' | 'disable', TranslationKey> = {
    enable: 'settings.workflowScheduledJobList.actions.enable',
    disable: 'settings.workflowScheduledJobList.actions.disable',
  }

  const toggleActionLabel = (enabled: boolean) =>
    i18nStore.t(enabled ? toggleActionKeys.disable : toggleActionKeys.enable)
</script>

{#if jobs.length === 0}
  <div
    class="border-border bg-card animate-fade-in-up rounded-xl border border-dashed px-4 py-14 text-center"
  >
    <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
      <CalendarClock class="text-muted-foreground size-5" />
    </div>
    <p class="text-foreground text-sm font-medium">
      {i18nStore.t('settings.workflowScheduledJobList.messages.emptyTitle')}
    </p>
    <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-sm">
      {i18nStore.t('settings.workflowScheduledJobList.messages.emptyDescription')}
    </p>
    <Button variant="outline" size="sm" class="mt-4" onclick={() => onNewJob?.()}>
      <Plus class="mr-1.5 size-3.5" />
      {i18nStore.t('settings.workflowScheduledJobList.actions.newJob')}
    </Button>
  </div>
{:else}
  <div class="space-y-2">
    <div class="flex justify-end">
      <Button variant="outline" size="sm" onclick={() => onNewJob?.()}>
        <Plus class="mr-1.5 size-3.5" />
        {i18nStore.t('settings.workflowScheduledJobList.actions.newJob')}
      </Button>
    </div>
    {#each jobs as job (job.id)}
      {@const busy = actionJobId === job.id}
      <div class="border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3">
        <!-- Toggle -->
        <Switch
          checked={job.is_enabled}
          disabled={busy}
          onCheckedChange={() => onToggleEnabled?.(job)}
          aria-label={`${toggleActionLabel(job.is_enabled)} ${job.name}`}
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
            <span>
              {i18nStore.t('settings.workflowScheduledJobList.labels.status')}
              {job.ticket_template.status
                ? ` ${job.ticket_template.status}`
                : ` ${i18nStore.t('settings.workflowScheduledJobList.status.unassigned')}`}
            </span>
            <span>
              {i18nStore.t('settings.workflowScheduledJobList.labels.next')}{' '}
              {job.next_run_at ? formatRelativeTime(job.next_run_at) : '—'}
            </span>
            {#if job.last_run_at}
              <span>
                {i18nStore.t('settings.workflowScheduledJobList.labels.last')}{' '}
                {formatRelativeTime(job.last_run_at)}
              </span>
            {/if}
          </div>
        </button>

        <!-- Actions -->
        <div class="flex shrink-0 items-center gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={i18nStore.t('settings.workflowScheduledJobList.actions.edit')}
            title={i18nStore.t('settings.workflowScheduledJobList.actions.edit')}
            onclick={() => onEditJob?.(job)}
          >
            <Pencil class="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={i18nStore.t('settings.workflowScheduledJobList.actions.runOnce')}
            title={i18nStore.t('settings.workflowScheduledJobList.actions.runOnce')}
            disabled={busy}
            onclick={() => onTriggerJob?.(job)}
          >
            <Play class="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={i18nStore.t('settings.workflowScheduledJobList.actions.delete')}
            title={i18nStore.t('settings.workflowScheduledJobList.actions.delete')}
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
