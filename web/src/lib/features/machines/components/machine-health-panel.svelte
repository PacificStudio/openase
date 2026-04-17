<script lang="ts">
  import { CircleCheck, CircleHelp, CircleX, GitBranch, Globe, Terminal } from '@lucide/svelte'
  import { Badge } from '$ui/badge'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type {
    MachineItem,
    MachineProbeResult,
    MachineReachabilityMode,
    MachineSnapshot,
  } from '../types'
  import {
    buildAuditRows,
    buildLevelCards,
    buildStatCards,
    checkedAtLabel,
    levelState,
    runtimeLabel,
    stateBadgeVariant,
    stateLabel,
    toTruthyState,
    type TruthyState,
  } from './machine-health-panel-view'
  import MachineGpuTable from './machine-gpu-table.svelte'
  import MachineHealthAuditPanel from './machine-health-audit-panel.svelte'
  import MachineHealthHeader from './machine-health-header.svelte'
  import MachineHealthRuntimeTable from './machine-health-runtime-table.svelte'
  import MachineHealthSetupPanel from './machine-health-setup-panel.svelte'
  import MachineProbeCard from './machine-probe-card.svelte'
  import { machineErrorMessage, runMachineSSHBootstrap } from './machines-page-api'

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
    <MachineHealthSetupPanel
      {setupGuide}
      {hasTrouble}
      {setupExpanded}
      {showBootstrapRepair}
      {bootstrapRunning}
      {bootstrapResult}
      {bootstrapError}
      onToggle={() => (setupExpanded = !setupExpanded)}
      onBootstrapRepair={handleBootstrapRepair}
    />

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
      <MachineHealthRuntimeTable
        {runtimeRows}
        checkedLabel={checkedAtLabel(snapshot.agentEnvironmentCheckedAt)}
        badgeVariant={stateBadgeVariant(l4State)}
        badgeLabel={stateLabel(l4State)}
        {truthyIcon}
        {truthyColorClass}
        {runtimeLabel}
        {toTruthyState}
      />
    {/if}

    {#if snapshot.gpus.length > 0}
      <MachineGpuTable {snapshot} />
    {/if}

    {#if snapshot.fullAudit}
      {@const l5State = levelState(snapshot.monitor.l5)}
      <MachineHealthAuditPanel
        {auditRows}
        checkedLabel={checkedAtLabel(snapshot.fullAudit.checkedAt)}
        badgeVariant={stateBadgeVariant(l5State)}
        badgeLabel={stateLabel(l5State)}
        {truthyIcon}
        {truthyColorClass}
        {auditRowIcons}
      />
    {/if}
  {/if}

  {#if probe}
    <MachineProbeCard {probe} />
  {/if}
</div>
