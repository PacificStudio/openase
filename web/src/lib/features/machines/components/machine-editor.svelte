<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import MachineHealthPanel from './machine-health-panel.svelte'
  import { isLocalMachine, machineStatusOptions } from '../model'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
  } from '../types'

  let {
    mode,
    machine,
    draft,
    snapshot,
    probe,
    loadingHealth = false,
    saving = false,
    testing = false,
    deleting = false,
    onDraftChange,
    onSave,
    onTest,
    onDelete,
    onReset,
  }: {
    mode: MachineEditorMode
    machine: MachineItem | null
    draft: MachineDraft
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loadingHealth?: boolean
    saving?: boolean
    testing?: boolean
    deleting?: boolean
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onSave?: () => void
    onTest?: () => void
    onDelete?: () => void
    onReset?: () => void
  } = $props()

  const localMachine = $derived(isLocalMachine(machine, draft))

  function updateField(field: MachineDraftField, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }
</script>

<div class="space-y-4">
  <div class="border-border bg-card rounded-2xl border">
    <div class="border-border flex flex-wrap items-start justify-between gap-3 border-b px-5 py-4">
      <div>
        <div class="flex items-center gap-2">
          <h2 class="text-foreground text-base font-semibold">
            {mode === 'create' ? 'Register remote machine' : (machine?.name ?? 'Machine')}
          </h2>
          {#if mode === 'edit' && machine}
            <Badge variant="outline" class="capitalize">{machine.status}</Badge>
          {/if}
        </div>
        <p class="text-muted-foreground mt-1 text-sm">
          {#if mode === 'create'}
            Organizations already seed the local runner. Use this form to register remote capacity.
          {:else}
            Configure SSH access, workspace paths, and runtime defaults for this machine.
          {/if}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" onclick={onReset} disabled={saving || testing || deleting}>
          Reset
        </Button>
        <Button
          variant="outline"
          onclick={onTest}
          disabled={mode === 'create' || testing || saving}
        >
          {testing ? 'Testing…' : 'Test connection'}
        </Button>
        <Button onclick={onSave} disabled={saving || deleting}>
          {saving ? 'Saving…' : mode === 'create' ? 'Create machine' : 'Save changes'}
        </Button>
        <Button
          variant="destructive"
          onclick={onDelete}
          disabled={mode === 'create' || localMachine || deleting || saving}
          title={localMachine ? 'The seeded local machine cannot be deleted.' : undefined}
        >
          {deleting ? 'Deleting…' : 'Delete'}
        </Button>
      </div>
    </div>

    <div class="space-y-5 px-5 py-5">
      {#if localMachine}
        <div class="border-info/40 bg-info/10 rounded-xl border px-4 py-3 text-sm">
          The local machine keeps its reserved identity. Name, host, and SSH settings stay fixed.
        </div>
      {/if}

      <div class="grid gap-4 md:grid-cols-2">
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

        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.status}
            onValueChange={(value) => onDraftChange?.('status', value || 'maintenance')}
          >
            <Select.Trigger class="w-full capitalize">{draft.status}</Select.Trigger>
            <Select.Content>
              {#each machineStatusOptions as status (status)}
                <Select.Item value={status} class="capitalize">{status}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

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
    </div>
  </div>

  <div class="border-border bg-card rounded-2xl border px-5 py-5">
    <MachineHealthPanel {machine} {snapshot} {probe} loading={loadingHealth} />
  </div>
</div>
