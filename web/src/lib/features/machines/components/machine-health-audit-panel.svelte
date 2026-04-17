<script lang="ts">
  import { Badge } from '$ui/badge'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { formatMachineRelativeTime } from '../machine-i18n'
  import type { HealthAuditRow, TruthyState } from './machine-health-panel-view'

  let {
    auditRows,
    checkedLabel,
    badgeVariant,
    badgeLabel,
    truthyIcon,
    truthyColorClass,
    auditRowIcons,
  }: {
    auditRows: HealthAuditRow[]
    checkedLabel: string
    badgeVariant: 'default' | 'secondary' | 'outline' | 'destructive'
    badgeLabel: string
    truthyIcon: Record<TruthyState, typeof import('@lucide/svelte').CircleCheck>
    truthyColorClass: Record<TruthyState, string>
    auditRowIcons: {
      git: typeof import('@lucide/svelte').GitBranch
      'gh-cli': typeof import('@lucide/svelte').Terminal
      network: typeof import('@lucide/svelte').Globe
    }
  } = $props()
</script>

<div class="border-border bg-card rounded-xl border">
  <div
    class="border-border flex flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
  >
    <div class="flex items-center gap-2">
      <h4 class="text-foreground text-sm font-semibold">
        {i18nStore.t('machines.machineHealthPanel.heading.toolingAudit')}
      </h4>
      <Badge variant={badgeVariant}>{badgeLabel}</Badge>
    </div>
    <span class="text-muted-foreground text-xs">{checkedLabel}</span>
  </div>
  <div class="divide-border divide-y">
    {#each auditRows as row (row.kind)}
      {@const RowIcon = auditRowIcons[row.kind]}
      <div class="flex items-start gap-3 px-4 py-3">
        <div class="bg-muted mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md">
          <RowIcon class="text-muted-foreground size-3.5" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-foreground text-sm font-medium">{row.label}</span>
            {#if row.kind === 'git' || row.kind === 'gh-cli'}
              {@const Icon = truthyIcon[row.installed]}
              <div class={`flex items-center gap-1 ${truthyColorClass[row.installed]}`}>
                <Icon class="size-3.5" />
                <span class="text-xs font-medium">
                  {row.installed === 'yes'
                    ? i18nStore.t('machines.machineHealthPanel.status.installed')
                    : row.installed === 'no'
                      ? i18nStore.t('machines.machineHealthPanel.status.missing')
                      : i18nStore.t('machines.machineHealthPanel.status.unknown')}
                </span>
              </div>
              {#if row.kind === 'gh-cli'}
                <Badge variant="outline" class="text-[10px]">
                  {i18nStore.t('machines.machineHealthPanel.badge.observational')}
                </Badge>
              {/if}
            {/if}
          </div>
          <div class="text-muted-foreground mt-1 text-xs">
            {#if row.kind === 'git'}
              {row.identity ?? i18nStore.t('machines.machineHealthPanel.hint.noGitIdentity')}
            {:else if row.kind === 'gh-cli'}
              {row.authStatus ?? i18nStore.t('machines.machineHealthPanel.hint.noAuthStatus')}
            {:else if row.kind === 'network'}
              <div class="mt-1 flex items-center gap-3">
                {#each row.endpoints as endpoint (endpoint.name)}
                  {@const EpIcon = truthyIcon[endpoint.reachable]}
                  <div class="flex items-center gap-1">
                    <EpIcon class={`size-3 ${truthyColorClass[endpoint.reachable]}`} />
                    <span>{endpoint.name}</span>
                  </div>
                {/each}
              </div>
              {#if row.auditTimestamp}
                <div class="mt-1 text-[11px]">
                  {i18nStore.t('machines.machineHealthPanel.hint.capturedAt', {
                    time: formatMachineRelativeTime(row.auditTimestamp),
                  })}
                </div>
              {/if}
            {/if}
          </div>
        </div>
      </div>
    {/each}
  </div>
</div>
