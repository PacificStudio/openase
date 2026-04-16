<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Input } from '$ui/input'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import MachineEditorGuidance from './machine-editor-guidance.svelte'
  import {
    getWorkspaceRootRecommendation,
    getWorkspaceRootState,
    isLocalMachine,
    machineReachabilityLabel,
    normalizeExecutionMode,
    normalizeReachabilityMode,
  } from '../model'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineItem,
    MachineReachabilityMode,
    WorkspaceRootState,
  } from '../types'
  import { ChevronDown, ChevronUp } from '@lucide/svelte'
  import { slide } from 'svelte/transition'

  let {
    machine,
    draft,
    onDraftChange,
  }: {
    machine: MachineItem | null
    draft: MachineDraft
    onDraftChange?: (field: MachineDraftField, value: string) => void
  } = $props()

  const localMachine = $derived(isLocalMachine(machine, draft))
  const reachabilityMode = $derived(normalizeReachabilityMode(draft.reachabilityMode, draft.host))
  const executionMode = $derived(normalizeExecutionMode(draft.executionMode, draft.host))

  // The full topology wizard is overwhelming for an already-created machine.
  // Collapse it by default for edit mode; expose a "change connection" toggle
  // so users can still alter the topology when they need to.
  let guidanceOpen = $state(false)
  $effect(() => {
    if (machine === null) guidanceOpen = true
  })
  const workspaceRootRecommendation = $derived(getWorkspaceRootRecommendation({ draft, machine }))
  const workspaceRootState = $derived(getWorkspaceRootState({ draft, machine }))

  const workspaceStateTone: Record<WorkspaceRootState['kind'], string> = {
    recommended: 'border-emerald-500/30 bg-emerald-500/12 text-emerald-700',
    saved: 'border-sky-500/30 bg-sky-500/12 text-sky-700',
    manual: 'border-amber-500/30 bg-amber-500/12 text-amber-700',
    empty: 'border-slate-500/20 bg-slate-500/10 text-slate-700',
  }

  function updateField(field: MachineDraftField, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }

  function updateReachability(mode: MachineReachabilityMode) {
    onDraftChange?.('reachabilityMode', mode)
  }
</script>

<div class="space-y-5">
  {#if localMachine}
    <div class="border-info/40 bg-info/10 rounded-lg border px-3.5 py-2.5 text-xs">
      {i18nStore.t('machines.machineEditor.warning.localIdentityReserved')}
    </div>
  {/if}

  {#if machine}
    <!-- Existing machine: hide the big guided wizard behind a disclosure. The
         summary row keeps the current topology visible without the noise. -->
    <div class="border-border bg-card rounded-lg border">
      <button
        type="button"
        class="hover:bg-muted/40 flex w-full items-center justify-between gap-3 rounded-lg px-3.5 py-2.5 text-left transition-colors"
        onclick={() => (guidanceOpen = !guidanceOpen)}
        aria-expanded={guidanceOpen}
        data-testid="machine-editor-guidance-toggle"
      >
        <div class="flex min-w-0 items-center gap-2">
          <span class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineEditor.connection.heading')}
          </span>
          <Badge variant="outline" class="text-[10px]">
            {machineReachabilityLabel(reachabilityMode)}
          </Badge>
        </div>
        <span class="text-muted-foreground flex items-center gap-1 text-[11px]">
          {guidanceOpen
            ? i18nStore.t('machines.machineEditor.connection.hide')
            : i18nStore.t('machines.machineEditor.connection.change')}
          {#if guidanceOpen}
            <ChevronUp class="size-3.5" />
          {:else}
            <ChevronDown class="size-3.5" />
          {/if}
        </span>
      </button>
      {#if guidanceOpen}
        <div class="border-border border-t px-3.5 py-3" transition:slide={{ duration: 200 }}>
          <MachineEditorGuidance {machine} {draft} onSelectReachability={updateReachability} />
        </div>
      {/if}
    </div>
  {:else}
    <MachineEditorGuidance {machine} {draft} onSelectReachability={updateReachability} />
  {/if}

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('machines.machineEditor.heading.identityAndConnection')}
    </h3>

    <div class="grid gap-3 md:grid-cols-3">
      <div class="space-y-1.5">
        <Label for="machine-name" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.name')}
        </Label>
        <Input
          id="machine-name"
          value={draft.name}
          disabled={localMachine}
          oninput={(event) => updateField('name', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-host" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.host')}
        </Label>
        <Input
          id="machine-host"
          value={draft.host}
          disabled={localMachine}
          oninput={(event) => updateField('host', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-port" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.port')}
        </Label>
        <Input
          id="machine-port"
          value={draft.port}
          oninput={(event) => updateField('port', event)}
        />
      </div>
    </div>

    {#if reachabilityMode === 'direct_connect' && executionMode === 'websocket'}
      <div class="space-y-1.5">
        <Label for="machine-advertised-endpoint" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.directConnectEndpoint')}
        </Label>
        <Input
          id="machine-advertised-endpoint"
          value={draft.advertisedEndpoint}
          placeholder="wss://builder.example.com/openase"
          oninput={(event) => updateField('advertisedEndpoint', event)}
        />
        <p class="text-muted-foreground text-[11px]">
          {i18nStore.t('machines.machineEditor.hints.directConnectListener')}
        </p>
      </div>
    {/if}
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('machines.machineEditor.heading.sshHelper')}
    </h3>

    {#if localMachine}
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('machines.machineEditor.helperAccess.localExecution')}
      </p>
    {:else}
      <div class="grid gap-3 md:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="machine-ssh-user" class="text-xs">
            {i18nStore.t('machines.machineEditor.labels.sshUser')}
          </Label>
          <Input
            id="machine-ssh-user"
            value={draft.sshUser}
            disabled={localMachine}
            oninput={(event) => updateField('sshUser', event)}
          />
        </div>

        <div class="space-y-1.5">
          <Label for="machine-ssh-key" class="text-xs">
            {i18nStore.t('machines.machineEditor.labels.sshKeyPath')}
          </Label>
          <Input
            id="machine-ssh-key"
            value={draft.sshKeyPath}
            disabled={localMachine}
            oninput={(event) => updateField('sshKeyPath', event)}
          />
        </div>
      </div>
      <p class="text-muted-foreground text-xs">
        {#if reachabilityMode === 'reverse_connect'}
          {i18nStore.t('machines.machineEditor.helperAccess.reverseConnect')}
        {:else}
          {i18nStore.t('machines.machineEditor.helperAccess.directConnect')}
        {/if}
      </p>
    {/if}
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('machines.machineEditor.heading.workspaceRuntime')}
    </h3>

    <div class="border-border bg-card flex items-center gap-3 rounded-lg border px-3.5 py-2.5">
      <div class="flex flex-wrap items-center gap-2">
        <code class="bg-muted text-foreground rounded px-1.5 py-0.5 text-xs">
          {workspaceRootRecommendation.value}
        </code>
        <Badge variant="outline" class={workspaceStateTone[workspaceRootState.kind]}>
          {workspaceRootState.label}
        </Badge>
      </div>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <div class="space-y-1.5">
        <Label for="machine-workspace-root" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.workspaceRoot')}
        </Label>
        <Input
          id="machine-workspace-root"
          value={draft.workspaceRoot}
          placeholder={workspaceRootRecommendation.value}
          oninput={(event) => updateField('workspaceRoot', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-agent-cli" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.agentCLIPath')}
        </Label>
        <Input
          id="machine-agent-cli"
          value={draft.agentCLIPath}
          oninput={(event) => updateField('agentCLIPath', event)}
        />
      </div>
    </div>
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('machines.machineEditor.heading.metadata')}
    </h3>

    <div class="space-y-1.5">
      <Label for="machine-description" class="text-xs">
        {i18nStore.t('machines.machineEditor.labels.description')}
      </Label>
      <Textarea
        id="machine-description"
        value={draft.description}
        rows={2}
        oninput={(event) => updateField('description', event)}
      />
    </div>

    <div class="grid gap-3 lg:grid-cols-2">
      <div class="space-y-1.5">
        <Label for="machine-labels" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.labels')}
        </Label>
        <Textarea
          id="machine-labels"
          value={draft.labels}
          rows={3}
          placeholder="gpu, a100, europe-west"
          oninput={(event) => updateField('labels', event)}
        />
        <p class="text-muted-foreground text-[11px]">
          {i18nStore.t('machines.machineEditor.hints.labelSeparator')}
        </p>
      </div>

      <div class="space-y-1.5">
        <Label for="machine-env-vars" class="text-xs">
          {i18nStore.t('machines.machineEditor.labels.environmentVariables')}
        </Label>
        <Textarea
          id="machine-env-vars"
          value={draft.envVars}
          rows={3}
          placeholder={`CUDA_VISIBLE_DEVICES=0\nOPENASE_AGENT_HOME=/srv/openase`}
          oninput={(event) => updateField('envVars', event)}
        />
        <p class="text-muted-foreground text-[11px]">
          {i18nStore.t('machines.machineEditor.hints.envVarsHelper')}
        </p>
      </div>
    </div>
  </section>
</div>
