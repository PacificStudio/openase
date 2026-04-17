<script lang="ts">
  import { Badge } from '$ui/badge'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { MachineCLIStatus } from '../types'
  import type { TruthyState } from './machine-health-panel-view'

  let {
    runtimeRows,
    checkedLabel,
    badgeVariant,
    badgeLabel,
    truthyIcon,
    truthyColorClass,
    runtimeLabel,
    toTruthyState,
  }: {
    runtimeRows: MachineCLIStatus[]
    checkedLabel: string
    badgeVariant: 'default' | 'secondary' | 'outline' | 'destructive'
    badgeLabel: string
    truthyIcon: Record<TruthyState, typeof import('@lucide/svelte').CircleCheck>
    truthyColorClass: Record<TruthyState, string>
    runtimeLabel: (runtime: MachineCLIStatus) => string
    toTruthyState: (value: boolean | undefined) => TruthyState
  } = $props()
</script>

<div class="border-border bg-card rounded-xl border">
  <div
    class="border-border flex flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
  >
    <div class="flex items-center gap-2">
      <h4 class="text-foreground text-sm font-semibold">
        {i18nStore.t('machines.machineHealthPanel.heading.runtimeProviders')}
      </h4>
      <Badge variant={badgeVariant}>{badgeLabel}</Badge>
    </div>
    <span class="text-muted-foreground text-xs">{checkedLabel}</span>
  </div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-border text-muted-foreground border-b text-left text-xs">
          <th class="px-4 py-2 font-medium">
            {i18nStore.t('machines.machineHealthPanel.tableColumns.runtime')}
          </th>
          <th class="px-4 py-2 font-medium">
            {i18nStore.t('machines.machineHealthPanel.tableColumns.installed')}
          </th>
          <th class="px-4 py-2 font-medium">
            {i18nStore.t('machines.machineHealthPanel.tableColumns.auth')}
          </th>
          <th class="px-4 py-2 font-medium">
            {i18nStore.t('machines.machineHealthPanel.tableColumns.ready')}
          </th>
          <th class="px-4 py-2 font-medium">
            {i18nStore.t('machines.machineHealthPanel.tableColumns.version')}
          </th>
        </tr>
      </thead>
      <tbody>
        {#each runtimeRows as runtime (runtime.name)}
          {@const installedState = toTruthyState(runtime.installed)}
          {@const readyState = toTruthyState(runtime.ready)}
          {@const InstalledIcon = truthyIcon[installedState]}
          {@const ReadyIcon = truthyIcon[readyState]}
          <tr class="border-border/60 border-b last:border-0">
            <td class="px-4 py-3 font-medium">{runtimeLabel(runtime)}</td>
            <td class="px-4 py-3">
              <InstalledIcon class={`size-4 ${truthyColorClass[installedState]}`} />
            </td>
            <td class="px-4 py-3 text-xs">
              {[runtime.authStatus, runtime.authMode].filter(Boolean).join(' · ') ||
                i18nStore.t('machines.machineHealthPanel.status.unknown')}
            </td>
            <td class="px-4 py-3">
              <ReadyIcon class={`size-4 ${truthyColorClass[readyState]}`} />
            </td>
            <td class="text-muted-foreground px-4 py-3 text-xs">
              {runtime.version ?? i18nStore.t('machines.machineHealthPanel.status.unknown')}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
