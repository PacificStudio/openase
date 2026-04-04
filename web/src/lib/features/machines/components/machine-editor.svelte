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
    normalizeConnectionMode,
  } from '../model'
  import type {
    MachineConnectionMode,
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    WorkspaceRootState,
  } from '../types'

  let {
    mode,
    machine,
    draft,
    onDraftChange,
  }: {
    mode: MachineEditorMode
    machine: MachineItem | null
    draft: MachineDraft
    onDraftChange?: (field: MachineDraftField, value: string) => void
  } = $props()

  const localMachine = $derived(isLocalMachine(machine, draft))
  const connectionMode = $derived(normalizeConnectionMode(draft.connectionMode, draft.host))
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

  function updateMode(mode: MachineConnectionMode) {
    onDraftChange?.('connectionMode', mode)
  }
</script>

<div class="space-y-6">
  {#if localMachine}
    <div class="border-info/40 bg-info/10 rounded-xl border px-4 py-3 text-sm">
      The local machine keeps its reserved identity. Name, host, and SSH settings stay fixed.
    </div>
  {/if}

  <MachineEditorGuidance {machine} {draft} onSelectMode={updateMode} />

  <section class="space-y-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Identity & network</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {mode === 'create'
          ? 'Register the machine identity, transport, and advertised endpoint.'
          : 'Update how OpenASE addresses this machine.'}
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <div class="space-y-2">
        <Label for="machine-name">Name</Label>
        <Input
          id="machine-name"
          value={draft.name}
          disabled={localMachine}
          oninput={(event) => updateField('name', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="machine-host">Host</Label>
        <Input
          id="machine-host"
          value={draft.host}
          disabled={localMachine}
          oninput={(event) => updateField('host', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="machine-port">Port</Label>
        <Input
          id="machine-port"
          value={draft.port}
          oninput={(event) => updateField('port', event)}
        />
      </div>
    </div>

    {#if connectionMode === 'ws_listener'}
      <div class="space-y-2">
        <Label for="machine-advertised-endpoint">Advertised listener endpoint</Label>
        <Input
          id="machine-advertised-endpoint"
          value={draft.advertisedEndpoint}
          placeholder="wss://builder.example.com/openase"
          oninput={(event) => updateField('advertisedEndpoint', event)}
        />
        <p class="text-muted-foreground text-xs">
          OpenASE connects to this machine-advertised websocket endpoint when running listener mode
          checks.
        </p>
      </div>
    {/if}
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Transport credentials</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Only the fields required by the selected connection mode stay editable here.
      </p>
    </div>

    {#if connectionMode === 'ssh'}
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label for="machine-ssh-user">SSH user</Label>
          <Input
            id="machine-ssh-user"
            value={draft.sshUser}
            disabled={localMachine}
            oninput={(event) => updateField('sshUser', event)}
          />
        </div>

        <div class="space-y-2">
          <Label for="machine-ssh-key">SSH key path</Label>
          <Input
            id="machine-ssh-key"
            value={draft.sshKeyPath}
            disabled={localMachine}
            oninput={(event) => updateField('sshKeyPath', event)}
          />
        </div>
      </div>
    {:else}
      <div class="border-border bg-card text-muted-foreground rounded-xl border px-4 py-3 text-sm">
        {#if connectionMode === 'local'}
          Local machines do not use SSH credentials.
        {:else if connectionMode === 'ws_reverse'}
          Reverse websocket machines rely on daemon registration instead of SSH credentials.
        {:else}
          Listener websocket machines use the advertised endpoint instead of SSH credentials.
        {/if}
      </div>
    {/if}
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Workspace & runtime</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Configure where OpenASE writes workspaces and which CLI path the machine should use.
      </p>
    </div>

    <div class="border-border bg-card space-y-3 rounded-xl border px-4 py-4">
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">Recommended root</Badge>
        <code class="bg-muted text-foreground rounded px-2 py-1 text-xs">
          {workspaceRootRecommendation.value}
        </code>
        <Badge variant="outline" class={workspaceStateTone[workspaceRootState.kind]}>
          {workspaceRootState.label}
        </Badge>
      </div>
      <p class="text-muted-foreground text-xs">{workspaceRootRecommendation.reason}</p>
      {#if localMachine}
        <p class="text-muted-foreground text-xs">
          The local `~/.openase/workspace` shortcut stays readable in the UI and is expanded to an
          absolute path when saved.
        </p>
      {/if}
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="machine-workspace-root">Workspace root</Label>
        <Input
          id="machine-workspace-root"
          value={draft.workspaceRoot}
          placeholder={workspaceRootRecommendation.value}
          oninput={(event) => updateField('workspaceRoot', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="machine-agent-cli">Agent CLI path</Label>
        <Input
          id="machine-agent-cli"
          value={draft.agentCLIPath}
          oninput={(event) => updateField('agentCLIPath', event)}
        />
      </div>
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Metadata</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Optional notes, routing labels, and environment variables.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="machine-description">Description</Label>
      <Textarea
        id="machine-description"
        value={draft.description}
        rows={3}
        oninput={(event) => updateField('description', event)}
      />
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="space-y-2">
        <Label for="machine-labels">Labels</Label>
        <Textarea
          id="machine-labels"
          value={draft.labels}
          rows={4}
          placeholder="gpu, a100, europe-west"
          oninput={(event) => updateField('labels', event)}
        />
        <p class="text-muted-foreground text-xs">Separate labels with commas or new lines.</p>
      </div>

      <div class="space-y-2">
        <Label for="machine-env-vars">Environment variables</Label>
        <Textarea
          id="machine-env-vars"
          value={draft.envVars}
          rows={4}
          placeholder={`CUDA_VISIBLE_DEVICES=0\nOPENASE_AGENT_HOME=/srv/openase`}
          oninput={(event) => updateField('envVars', event)}
        />
        <p class="text-muted-foreground text-xs">One `KEY=VALUE` pair per line.</p>
      </div>
    </div>
  </section>
</div>
