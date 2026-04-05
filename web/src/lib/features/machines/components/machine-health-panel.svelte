<script lang="ts">
  import { CircleCheck, CircleX, CircleHelp, GitBranch, Terminal, Globe } from '@lucide/svelte'
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import type { MachineItem, MachineProbeResult, MachineSnapshot } from '../types'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type { TruthyState } from './machine-health-panel-view'
  import {
    buildAuditRows,
    buildStatCards,
    checkedAtLabel,
    runtimeLabel,
    stateBadgeVariant,
    stateLabel,
    toTruthyState,
    levelState,
  } from './machine-health-panel-view'
  import MachineGpuTable from './machine-gpu-table.svelte'
  import MachineHealthHeader from './machine-health-header.svelte'
  import MachineProbeCard from './machine-probe-card.svelte'

  const truthyIcon: Record<TruthyState, typeof CircleCheck> = {
    yes: CircleCheck,
    no: CircleX,
    unknown: CircleHelp,
  }

  const truthyColorClass: Record<TruthyState, string> = {
    yes: 'text-emerald-500',
    no: 'text-destructive',
    unknown: 'text-muted-foreground',
  }

  const auditRowIcons = {
    git: GitBranch,
    'gh-cli': Terminal,
    network: Globe,
  } as const

  let {
    machine,
    snapshot,
    probe,
    loading = false,
    refreshing = false,
    onRefresh,
  }: {
    machine: MachineItem | null
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loading?: boolean
    refreshing?: boolean
    onRefresh?: () => void
  } = $props()

  const statCards = $derived(snapshot ? buildStatCards(snapshot) : [])
  const runtimeRows = $derived(snapshot?.agentEnvironment ?? [])
  const auditRows = $derived(snapshot ? buildAuditRows(snapshot) : [])
  const setupGuide = $derived(buildMachineSetupGuide({ machine, snapshot }))
</script>

<div class="space-y-4">
  <MachineHealthHeader {machine} {snapshot} {loading} {refreshing} {onRefresh} />

  {#if !snapshot}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-8 text-center text-sm"
    >
      No monitor snapshot is available for this machine yet.
    </div>
  {:else}
    <div class="border-border bg-card rounded-xl border">
      <div class="border-border flex items-center justify-between border-b px-4 py-3">
        <div>
          <h4 class="text-foreground text-sm font-semibold">Setup guidance</h4>
          <p class="text-muted-foreground mt-1 text-xs">{setupGuide.topologySummary}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">{setupGuide.topologyLabel}</Badge>
          <Badge variant="outline">{setupGuide.stateLabel}</Badge>
        </div>
      </div>

      <div class="grid gap-3 px-4 py-4 lg:grid-cols-3">
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">{setupGuide.runtimeLabel}</p>
          <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.runtimeSummary}</p>
        </div>
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">{setupGuide.helperLabel}</p>
          <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.helperSummary}</p>
        </div>
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">Next steps</p>
          <ul class="text-muted-foreground space-y-1.5 text-xs leading-relaxed">
            {#each setupGuide.nextSteps as step, index (`${step}-${index}`)}
              <li>{step}</li>
            {/each}
          </ul>
        </div>
      </div>

      {#if setupGuide.commands.length > 0}
        <div class="border-border grid gap-3 border-t px-4 py-4">
          {#each setupGuide.commands as command (command.title)}
            <div class="rounded-lg border border-dashed px-3.5 py-3">
              <p class="text-foreground text-sm font-medium">{command.title}</p>
              <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
                {command.description}
              </p>
              <pre
                class="bg-muted/60 text-foreground mt-3 overflow-x-auto rounded-md px-3 py-2 text-xs whitespace-pre-wrap">{command.command}</pre>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      {#each statCards as card (card.label)}
        <div class="border-border bg-card rounded-xl border px-4 py-3">
          <p class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">{card.label}</p>
          <p class="text-foreground mt-2 text-sm font-semibold">{card.value}</p>
          <p class="text-muted-foreground mt-1 text-xs">{card.meta}</p>
        </div>
      {/each}
    </div>

    {#if snapshot.monitorErrors.length > 0}
      <div class="border-destructive/40 bg-destructive/10 rounded-xl border px-4 py-3">
        <p class="text-destructive text-sm font-medium">Monitor warnings</p>
        <ul class="text-destructive mt-2 space-y-1 text-xs">
          {#each snapshot.monitorErrors as error, index (`${error}-${index}`)}
            <li>{error}</li>
          {/each}
        </ul>
      </div>
    {/if}

    {#if runtimeRows.length > 0}
      {@const l4State = levelState(snapshot.monitor.l4)}
      <div class="border-border bg-card rounded-xl border">
        <div class="border-border flex items-center justify-between border-b px-4 py-3">
          <div class="flex items-center gap-2">
            <h4 class="text-foreground text-sm font-semibold">Runtime providers</h4>
            <Badge variant={stateBadgeVariant(l4State)}>{stateLabel(l4State)}</Badge>
          </div>
          <span class="text-muted-foreground text-xs">
            {checkedAtLabel(snapshot.agentEnvironmentCheckedAt)}
          </span>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-border text-muted-foreground border-b text-left text-xs">
                <th class="px-4 py-2 font-medium">Runtime</th>
                <th class="px-4 py-2 font-medium">Installed</th>
                <th class="px-4 py-2 font-medium">Auth</th>
                <th class="px-4 py-2 font-medium">Ready</th>
                <th class="px-4 py-2 font-medium">Version</th>
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
                    <InstalledIcon class="size-4 {truthyColorClass[installedState]}" />
                  </td>
                  <td class="px-4 py-3 text-xs">
                    {[runtime.authStatus, runtime.authMode].filter(Boolean).join(' · ') ||
                      'Unknown'}
                  </td>
                  <td class="px-4 py-3">
                    <ReadyIcon class="size-4 {truthyColorClass[readyState]}" />
                  </td>
                  <td class="text-muted-foreground px-4 py-3 text-xs"
                    >{runtime.version ?? 'Unknown'}</td
                  >
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}

    {#if snapshot.gpus.length > 0}
      <MachineGpuTable {snapshot} />
    {/if}

    {#if snapshot.fullAudit}
      {@const l5State = levelState(snapshot.monitor.l5)}
      <div class="border-border bg-card rounded-xl border">
        <div class="border-border flex items-center justify-between border-b px-4 py-3">
          <div class="flex items-center gap-2">
            <h4 class="text-foreground text-sm font-semibold">Tooling audit</h4>
            <Badge variant={stateBadgeVariant(l5State)}>{stateLabel(l5State)}</Badge>
          </div>
          <span class="text-muted-foreground text-xs">
            {checkedAtLabel(snapshot.fullAudit.checkedAt)}
          </span>
        </div>
        <div class="divide-border divide-y">
          {#each auditRows as row (row.kind)}
            {@const RowIcon = auditRowIcons[row.kind]}
            <div class="flex items-start gap-3 px-4 py-3">
              <div
                class="bg-muted mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md"
              >
                <RowIcon class="text-muted-foreground size-3.5" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-foreground text-sm font-medium">{row.label}</span>
                  {#if row.kind === 'git'}
                    {@const Icon = truthyIcon[row.installed]}
                    <div class="flex items-center gap-1 {truthyColorClass[row.installed]}">
                      <Icon class="size-3.5" />
                      <span class="text-xs font-medium">
                        {row.installed === 'yes'
                          ? 'Installed'
                          : row.installed === 'no'
                            ? 'Missing'
                            : 'Unknown'}
                      </span>
                    </div>
                  {:else if row.kind === 'gh-cli'}
                    {@const Icon = truthyIcon[row.installed]}
                    <div class="flex items-center gap-1 {truthyColorClass[row.installed]}">
                      <Icon class="size-3.5" />
                      <span class="text-xs font-medium">
                        {row.installed === 'yes'
                          ? 'Installed'
                          : row.installed === 'no'
                            ? 'Missing'
                            : 'Unknown'}
                      </span>
                    </div>
                    <Badge variant="outline" class="text-[10px]">Observational</Badge>
                  {/if}
                </div>
                <div class="text-muted-foreground mt-1 text-xs">
                  {#if row.kind === 'git'}
                    {row.identity ?? 'No git identity recorded'}
                  {:else if row.kind === 'gh-cli'}
                    {row.authStatus ?? 'No auth status recorded'}
                  {:else if row.kind === 'network'}
                    <div class="mt-1 flex items-center gap-3">
                      {#each row.endpoints as endpoint (endpoint.name)}
                        {@const EpIcon = truthyIcon[endpoint.reachable]}
                        <div class="flex items-center gap-1">
                          <EpIcon class="size-3 {truthyColorClass[endpoint.reachable]}" />
                          <span>{endpoint.name}</span>
                        </div>
                      {/each}
                    </div>
                    {#if row.auditTimestamp}
                      <div class="mt-1 text-[11px]">
                        Captured {formatRelativeTime(row.auditTimestamp)}
                      </div>
                    {/if}
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  {#if probe}
    <MachineProbeCard {probe} />
  {/if}
</div>
