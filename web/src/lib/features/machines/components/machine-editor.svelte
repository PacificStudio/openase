<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import MachineEditorGuidance from './machine-editor-guidance.svelte'
  import {
    getWorkspaceRootRecommendation,
    getWorkspaceRootState,
    isLocalMachine,
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
      Local machine identity is reserved. Name, host, and helper access settings are fixed.
    </div>
  {/if}

  <MachineEditorGuidance {machine} {draft} onSelectReachability={updateReachability} />

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">Identity & connection</h3>

    <div class="grid gap-3 md:grid-cols-3">
      <div class="space-y-1.5">
        <Label for="machine-name" class="text-xs">Name</Label>
        <Input
          id="machine-name"
          value={draft.name}
          disabled={localMachine}
          oninput={(event) => updateField('name', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-host" class="text-xs">Host</Label>
        <Input
          id="machine-host"
          value={draft.host}
          disabled={localMachine}
          oninput={(event) => updateField('host', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-port" class="text-xs">Port</Label>
        <Input
          id="machine-port"
          value={draft.port}
          oninput={(event) => updateField('port', event)}
        />
      </div>
    </div>

    {#if reachabilityMode === 'direct_connect' && executionMode === 'websocket'}
      <div class="space-y-1.5">
        <Label for="machine-advertised-endpoint" class="text-xs"
          >Direct-connect listener endpoint</Label
        >
        <Input
          id="machine-advertised-endpoint"
          value={draft.advertisedEndpoint}
          placeholder="wss://builder.example.com/openase"
          oninput={(event) => updateField('advertisedEndpoint', event)}
        />
        <p class="text-muted-foreground text-[11px]">
          OpenASE dials this websocket listener when the control plane can reach the machine
          directly.
        </p>
      </div>
    {/if}
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">SSH helper lane</h3>

    {#if localMachine}
      <p class="text-muted-foreground text-xs">Local execution does not use SSH helper access.</p>
    {:else}
      <div class="grid gap-3 md:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="machine-ssh-user" class="text-xs">SSH user</Label>
          <Input
            id="machine-ssh-user"
            value={draft.sshUser}
            disabled={localMachine}
            oninput={(event) => updateField('sshUser', event)}
          />
        </div>

        <div class="space-y-1.5">
          <Label for="machine-ssh-key" class="text-xs">SSH key path</Label>
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
          Optional helper access for assisted daemon bootstrap, diagnostics, or emergency repair.
          Runtime execution stays on the reverse-connect daemon.
        {:else}
          Optional helper access for quick bootstrap, diagnostics, or emergency repair. Runtime
          execution stays on the direct-connect listener above.
        {/if}
      </p>
    {/if}
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">Workspace & runtime</h3>

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
        <Label for="machine-workspace-root" class="text-xs">Workspace root</Label>
        <Input
          id="machine-workspace-root"
          value={draft.workspaceRoot}
          placeholder={workspaceRootRecommendation.value}
          oninput={(event) => updateField('workspaceRoot', event)}
        />
      </div>

      <div class="space-y-1.5">
        <Label for="machine-agent-cli" class="text-xs">Agent CLI path</Label>
        <Input
          id="machine-agent-cli"
          value={draft.agentCLIPath}
          oninput={(event) => updateField('agentCLIPath', event)}
        />
      </div>
    </div>
  </section>

  <section class="border-border space-y-3 border-t pt-5">
    <h3 class="text-foreground text-sm font-semibold">Metadata</h3>

    <div class="space-y-1.5">
      <Label for="machine-description" class="text-xs">Description</Label>
      <Textarea
        id="machine-description"
        value={draft.description}
        rows={2}
        oninput={(event) => updateField('description', event)}
      />
    </div>

    <div class="grid gap-3 lg:grid-cols-2">
      <div class="space-y-1.5">
        <Label for="machine-labels" class="text-xs">Labels</Label>
        <Textarea
          id="machine-labels"
          value={draft.labels}
          rows={3}
          placeholder="gpu, a100, europe-west"
          oninput={(event) => updateField('labels', event)}
        />
        <p class="text-muted-foreground text-[11px]">Comma or newline separated.</p>
      </div>

      <div class="space-y-1.5">
        <Label for="machine-env-vars" class="text-xs">Environment variables</Label>
        <Textarea
          id="machine-env-vars"
          value={draft.envVars}
          rows={3}
          placeholder={`CUDA_VISIBLE_DEVICES=0\nOPENASE_AGENT_HOME=/srv/openase`}
          oninput={(event) => updateField('envVars', event)}
        />
        <p class="text-muted-foreground text-[11px]">One KEY=VALUE per line.</p>
      </div>
    </div>
  </section>
</div>
