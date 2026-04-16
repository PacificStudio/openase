<!-- eslint-disable max-lines -->
<script lang="ts">
  import {
    CircleCheck,
    CircleX,
    CircleHelp,
    GitBranch,
    Terminal,
    Globe,
    ChevronDown,
    ChevronUp,
    Loader2,
    Wrench,
  } from '@lucide/svelte'
  import { slide } from 'svelte/transition'
  import { formatMachineRelativeTime } from '../machine-i18n'
  import { Badge } from '$ui/badge'
  import type {
    MachineItem,
    MachineProbeResult,
    MachineReachabilityMode,
    MachineSnapshot,
  } from '../types'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type { TruthyState } from './machine-health-panel-view'
  import {
    buildAuditRows,
    buildLevelCards,
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
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { runMachineSSHBootstrap, machineErrorMessage } from './machines-page-api'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'

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
  const levelCards = $derived(
    snapshot
      ? buildLevelCards(snapshot, machine?.reachability_mode as MachineReachabilityMode | undefined)
      : [],
  )
  const runtimeRows = $derived(snapshot?.agentEnvironment ?? [])
  const auditRows = $derived(snapshot ? buildAuditRows(snapshot) : [])
  const setupGuide = $derived(buildMachineSetupGuide({ machine, snapshot }))

  // When the machine is healthy we don't want the wall of setup commands and
  // topology prose in the user's face — that content is only useful as a
  // recovery aid. Auto-expand on signs of trouble; otherwise keep it tucked
  // behind a disclosure the user can open on demand.
  const hasTrouble = $derived(
    (snapshot?.monitorErrors?.length ?? 0) > 0 ||
      levelCards.some((card) => card.state === 'error') ||
      (machine?.reachability_mode === 'reverse_connect' &&
        Boolean(machine?.daemon_status) &&
        machine?.daemon_status?.session_state !== 'connected'),
  )
  let setupExpanded = $state(false)
  $effect(() => {
    if (hasTrouble) setupExpanded = true
  })

  const repairBootstrapTopology = $derived.by(() => {
    if (!machine?.id || !machine?.ssh_helper_enabled) return null
    if (machine.reachability_mode === 'reverse_connect') {
      return 'reverse-connect'
    }
    if (machine.reachability_mode === 'direct_connect' && machine.execution_mode === 'websocket') {
      return 'remote-listener'
    }
    return null
  })
  const showBootstrapRepair = $derived(Boolean(hasTrouble && repairBootstrapTopology))
  let bootstrapRunning = $state(false)
  let bootstrapResult = $state<MachineSSHBootstrapResult | null>(null)
  let bootstrapError = $state('')

  async function handleBootstrapRepair() {
    if (!machine?.id || !repairBootstrapTopology) return
    bootstrapRunning = true
    bootstrapResult = null
    bootstrapError = ''
    try {
      bootstrapResult = await runMachineSSHBootstrap(machine.id, {
        topology: repairBootstrapTopology,
      })
      toastStore.success(
        bootstrapResult.summary ||
          i18nStore.t('machines.machineHealthPanel.bootstrapRepair.successFallback'),
      )
      onRefresh?.()
    } catch (caughtError) {
      bootstrapError = machineErrorMessage(
        caughtError,
        i18nStore.t('machines.machineHealthPanel.bootstrapRepair.failureFallback'),
      )
      toastStore.error(bootstrapError)
    } finally {
      bootstrapRunning = false
    }
  }
</script>

<div class="space-y-4">
  <MachineHealthHeader {machine} {snapshot} {loading} {refreshing} {onRefresh} />

  {#if !snapshot}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-8 text-center text-sm"
    >
      {i18nStore.t('machines.machineHealthPanel.emptyState')}
    </div>
  {:else}
    <!-- Collapsed disclosure: one-line row showing topology + a toggle.
         Auto-expands when hasTrouble is true so problems are never hidden. -->
    <div class="border-border bg-card rounded-xl border">
      <button
        type="button"
        class="hover:bg-muted/40 flex w-full items-center justify-between gap-3 rounded-xl px-4 py-2.5 text-left transition-colors"
        onclick={() => (setupExpanded = !setupExpanded)}
        aria-expanded={setupExpanded}
        data-testid="machine-health-setup-toggle"
      >
        <div class="flex min-w-0 items-center gap-2">
          <span class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineHealthPanel.heading.setupGuidance')}
          </span>
          <Badge variant="outline" class="text-[10px]">{setupGuide.topologyLabel}</Badge>
          {#if hasTrouble}
            <Badge variant="destructive" class="text-[10px]">
              {i18nStore.t('machines.machineHealthPanel.status.needsAttention')}
            </Badge>
          {/if}
        </div>
        {#if setupExpanded}
          <ChevronUp class="text-muted-foreground size-4" />
        {:else}
          <ChevronDown class="text-muted-foreground size-4" />
        {/if}
      </button>

      {#if setupExpanded}
        <div class="border-border border-t" transition:slide={{ duration: 200 }}>
          <div class="flex flex-wrap items-center justify-between gap-2 px-4 py-3">
            <Badge variant="outline">{setupGuide.stateLabel}</Badge>
            {#if showBootstrapRepair}
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={bootstrapRunning}
                onclick={handleBootstrapRepair}
                data-testid="machine-health-bootstrap-repair"
              >
                {#if bootstrapRunning}
                  <Loader2 class="size-3.5 animate-spin" />
                  {i18nStore.t('machines.machineHealthPanel.bootstrapRepair.running')}
                {:else}
                  <Wrench class="size-3.5" />
                  {i18nStore.t('machines.machineHealthPanel.bootstrapRepair.action')}
                {/if}
              </Button>
            {/if}
          </div>
          <p class="text-muted-foreground px-4 pb-3 text-xs leading-relaxed">
            {hasTrouble && showBootstrapRepair
              ? i18nStore.t('machines.machineHealthPanel.bootstrapRepair.description')
              : setupGuide.topologySummary}
          </p>

          {#if bootstrapError}
            <div
              class="border-destructive/20 bg-destructive/5 text-destructive mx-4 mb-3 rounded-md border px-3 py-2 text-[11px] leading-relaxed"
            >
              {bootstrapError}
            </div>
          {/if}

          {#if bootstrapResult}
            <div
              class="border-primary/20 bg-primary/5 mx-4 mb-3 rounded-md border px-3 py-2 text-[11px] leading-relaxed"
            >
              <div class="flex flex-wrap gap-x-4 gap-y-1">
                <span>
                  <span class="text-muted-foreground">
                    {i18nStore.t(
                      'machines.machineEditorGuidance.progressive.bootstrap.serviceManager',
                    )}
                  </span>
                  <span class="text-foreground ml-1">{bootstrapResult.service_manager}</span>
                </span>
                <span>
                  <span class="text-muted-foreground">
                    {i18nStore.t(
                      'machines.machineEditorGuidance.progressive.bootstrap.serviceName',
                    )}
                  </span>
                  <span class="text-foreground ml-1">{bootstrapResult.service_name}</span>
                </span>
                <span>
                  <span class="text-muted-foreground">
                    {i18nStore.t(
                      'machines.machineEditorGuidance.progressive.bootstrap.serviceStatus',
                    )}
                  </span>
                  <span class="text-foreground ml-1">{bootstrapResult.service_status}</span>
                </span>
              </div>
              {#if bootstrapResult.connection_target}
                <p class="text-muted-foreground mt-1">
                  <span>
                    {i18nStore.t(
                      'machines.machineEditorGuidance.progressive.bootstrap.connectionTarget',
                    )}
                  </span>
                  <span class="text-foreground ml-1">{bootstrapResult.connection_target}</span>
                </p>
              {/if}
            </div>
          {/if}

          <div class="grid gap-3 px-4 py-3 lg:grid-cols-3">
            <div class="space-y-1.5">
              <p class="text-foreground text-sm font-medium">{setupGuide.runtimeLabel}</p>
              <p class="text-muted-foreground text-xs leading-relaxed">
                {setupGuide.runtimeSummary}
              </p>
            </div>
            <div class="space-y-1.5">
              <p class="text-foreground text-sm font-medium">{setupGuide.helperLabel}</p>
              <p class="text-muted-foreground text-xs leading-relaxed">
                {setupGuide.helperSummary}
              </p>
            </div>
            <div class="space-y-1.5">
              <p class="text-foreground text-sm font-medium">
                {i18nStore.t('machines.machineHealthPanel.heading.nextSteps')}
              </p>
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
      {/if}
    </div>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {#each statCards as card (card.label)}
        <div class="border-border bg-card rounded-xl border px-4 py-3">
          <p class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">{card.label}</p>
          <p class="text-foreground mt-2 text-sm font-semibold">{card.value}</p>
          <p class="text-muted-foreground mt-1 text-xs">{card.meta}</p>
        </div>
      {/each}
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      {#each levelCards as card (card.id)}
        <div class="border-border bg-card rounded-xl border px-4 py-3">
          <div class="flex items-center justify-between gap-2">
            <p class="text-foreground text-sm font-medium">{card.label}</p>
            <Badge variant={stateBadgeVariant(card.state)}>{stateLabel(card.state)}</Badge>
          </div>
          <p class="text-foreground mt-2 text-sm font-semibold">{card.value}</p>
          <p class="text-muted-foreground mt-1 text-xs">{card.meta}</p>
        </div>
      {/each}
    </div>

    {#if snapshot.monitorErrors.length > 0}
      <div class="border-destructive/40 bg-destructive/10 rounded-xl border px-4 py-3">
        <p class="text-destructive text-sm font-medium">
          {i18nStore.t('machines.machineHealthPanel.heading.monitorWarnings')}
        </p>
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
        <div
          class="border-border flex flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex items-center gap-2">
            <h4 class="text-foreground text-sm font-semibold">
              {i18nStore.t('machines.machineHealthPanel.heading.runtimeProviders')}
            </h4>
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
                    <InstalledIcon class="size-4 {truthyColorClass[installedState]}" />
                  </td>
                  <td class="px-4 py-3 text-xs">
                    {[runtime.authStatus, runtime.authMode].filter(Boolean).join(' · ') ||
                      i18nStore.t('machines.machineHealthPanel.status.unknown')}
                  </td>
                  <td class="px-4 py-3">
                    <ReadyIcon class="size-4 {truthyColorClass[readyState]}" />
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
    {/if}

    {#if snapshot.gpus.length > 0}
      <MachineGpuTable {snapshot} />
    {/if}

    {#if snapshot.fullAudit}
      {@const l5State = levelState(snapshot.monitor.l5)}
      <div class="border-border bg-card rounded-xl border">
        <div
          class="border-border flex flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex items-center gap-2">
            <h4 class="text-foreground text-sm font-semibold">
              {i18nStore.t('machines.machineHealthPanel.heading.toolingAudit')}
            </h4>
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
                          ? i18nStore.t('machines.machineHealthPanel.status.installed')
                          : row.installed === 'no'
                            ? i18nStore.t('machines.machineHealthPanel.status.missing')
                            : i18nStore.t('machines.machineHealthPanel.status.unknown')}
                      </span>
                    </div>
                  {:else if row.kind === 'gh-cli'}
                    {@const Icon = truthyIcon[row.installed]}
                    <div class="flex items-center gap-1 {truthyColorClass[row.installed]}">
                      <Icon class="size-3.5" />
                      <span class="text-xs font-medium">
                        {row.installed === 'yes'
                          ? i18nStore.t('machines.machineHealthPanel.status.installed')
                          : row.installed === 'no'
                            ? i18nStore.t('machines.machineHealthPanel.status.missing')
                            : i18nStore.t('machines.machineHealthPanel.status.unknown')}
                      </span>
                    </div>
                    <Badge variant="outline" class="text-[10px]">
                      {i18nStore.t('machines.machineHealthPanel.badge.observational')}
                    </Badge>
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
                          <EpIcon class="size-3 {truthyColorClass[endpoint.reachable]}" />
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
    {/if}
  {/if}

  {#if probe}
    <MachineProbeCard {probe} />
  {/if}
</div>
