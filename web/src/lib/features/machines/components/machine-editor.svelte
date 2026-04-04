<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { isLocalMachine } from '../model'
  import type { MachineDraft, MachineDraftField, MachineEditorMode, MachineItem } from '../types'

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
  const effectiveConnectionMode = $derived(localMachine ? 'local' : draft.connectionMode)

  function updateField(field: MachineDraftField, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }
</script>

<div class="space-y-6">
  {#if localMachine}
    <div class="border-info/40 bg-info/10 rounded-xl border px-4 py-3 text-sm">
      The local machine keeps its reserved identity. Name, host, and SSH settings stay fixed.
    </div>
  {/if}

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

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label>Transport</Label>
        <Select.Root
          type="single"
          value={effectiveConnectionMode}
          onValueChange={(value) => onDraftChange?.('connectionMode', value || 'ssh')}
          disabled={localMachine}
        >
          <Select.Trigger class="w-full">{effectiveConnectionMode}</Select.Trigger>
          <Select.Content>
            <Select.Item value="local">local</Select.Item>
            <Select.Item value="ssh">ssh</Select.Item>
            <Select.Item value="ws_listener">ws_listener</Select.Item>
            <Select.Item value="ws_reverse">ws_reverse</Select.Item>
          </Select.Content>
        </Select.Root>
      </div>

      {#if effectiveConnectionMode === 'ws_listener'}
        <div class="space-y-2">
          <Label for="machine-advertised-endpoint">Advertised websocket endpoint</Label>
          <Input
            id="machine-advertised-endpoint"
            value={draft.advertisedEndpoint}
            placeholder="wss://machine.example.com/openase/transport"
            oninput={(event) => updateField('advertisedEndpoint', event)}
          />
        </div>
      {/if}
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">SSH access</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Credentials used when the machine transport is SSH.
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="machine-ssh-user">SSH user</Label>
        <Input
          id="machine-ssh-user"
          value={draft.sshUser}
          disabled={localMachine || effectiveConnectionMode !== 'ssh'}
          oninput={(event) => updateField('sshUser', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="machine-ssh-key">SSH key path</Label>
        <Input
          id="machine-ssh-key"
          value={draft.sshKeyPath}
          disabled={localMachine || effectiveConnectionMode !== 'ssh'}
          oninput={(event) => updateField('sshKeyPath', event)}
        />
      </div>
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Workspace & runtime</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Configure where OpenASE writes workspaces and which CLI path the machine should use.
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="machine-workspace-root">Workspace root</Label>
        <Input
          id="machine-workspace-root"
          value={draft.workspaceRoot}
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
