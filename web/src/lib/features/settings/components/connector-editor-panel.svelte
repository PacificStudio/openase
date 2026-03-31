<script lang="ts">
  import type { ConnectorDraft } from '../connectors-model'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'

  const selectClasses =
    'border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2'

  let {
    editorMode = 'create',
    draft,
    saving = false,
    onDraftChange,
    onSave,
    onReset,
  }: {
    editorMode?: 'create' | 'edit'
    draft: ConnectorDraft
    saving?: boolean
    onDraftChange?: (field: keyof ConnectorDraft, value: string) => void
    onSave?: () => void
    onReset?: () => void
  } = $props()
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>{editorMode === 'create' ? 'Add connector' : 'Edit connector'}</Card.Title>
    <Card.Description>
      {editorMode === 'create'
        ? 'Create a project-scoped GitHub issue connector.'
        : 'Update connector policy, credentials, and runtime behavior without leaving Settings.'}
    </Card.Description>
  </Card.Header>
  <Card.Content class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2">
      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-name">Name</label>
        <Input
          id="connector-name"
          value={draft.name}
          oninput={(event) => onDraftChange?.('name', event.currentTarget.value)}
          placeholder="GitHub Backend"
        />
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-project-ref">Project ref</label>
        <Input
          id="connector-project-ref"
          value={draft.projectRef}
          oninput={(event) => onDraftChange?.('projectRef', event.currentTarget.value)}
          placeholder="acme/backend"
        />
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-status">Status</label>
        <select
          id="connector-status"
          class={selectClasses}
          value={draft.status}
          onchange={(event) => onDraftChange?.('status', event.currentTarget.value)}
        >
          <option value="active">active</option>
          <option value="paused">paused</option>
          <option value="error">error</option>
        </select>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-sync-direction">Sync direction</label>
        <select
          id="connector-sync-direction"
          class={selectClasses}
          value={draft.syncDirection}
          onchange={(event) => onDraftChange?.('syncDirection', event.currentTarget.value)}
        >
          <option value="bidirectional">bidirectional</option>
          <option value="pull_only">pull_only</option>
          <option value="push_only">push_only</option>
        </select>
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-poll-interval">Poll interval</label>
        <Input
          id="connector-poll-interval"
          value={draft.pollInterval}
          oninput={(event) => onDraftChange?.('pollInterval', event.currentTarget.value)}
          placeholder="5m"
        />
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-base-url">Base URL</label>
        <Input
          id="connector-base-url"
          value={draft.baseURL}
          oninput={(event) => onDraftChange?.('baseURL', event.currentTarget.value)}
          placeholder="https://api.github.com"
        />
      </div>
    </div>

    <div class="space-y-2">
      <label class="text-sm font-medium" for="connector-label-filter">Label filter</label>
      <Input
        id="connector-label-filter"
        value={draft.labelFilter}
        oninput={(event) => onDraftChange?.('labelFilter', event.currentTarget.value)}
        placeholder="openase, triage"
      />
    </div>

    <div class="space-y-2">
      <label class="text-sm font-medium" for="connector-status-mapping">Status mapping</label>
      <Textarea
        id="connector-status-mapping"
        value={draft.statusMapping}
        oninput={(event) => onDraftChange?.('statusMapping', event.currentTarget.value)}
        rows={4}
        placeholder={'open=Todo\nclosed=Done'}
      />
    </div>

    <div class="grid gap-4 sm:grid-cols-2">
      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-auth-token">
          {editorMode === 'create' ? 'Auth token' : 'Replace auth token'}
        </label>
        <Input
          id="connector-auth-token"
          type="password"
          value={draft.authToken}
          oninput={(event) => onDraftChange?.('authToken', event.currentTarget.value)}
          placeholder={editorMode === 'create' ? 'ghu_...' : 'Leave blank to keep current token'}
        />
      </div>

      <div class="space-y-2">
        <label class="text-sm font-medium" for="connector-webhook-secret">
          {editorMode === 'create' ? 'Webhook secret' : 'Rotate webhook secret'}
        </label>
        <Input
          id="connector-webhook-secret"
          type="password"
          value={draft.webhookSecret}
          oninput={(event) => onDraftChange?.('webhookSecret', event.currentTarget.value)}
          placeholder={editorMode === 'create'
            ? 'shared-secret'
            : 'Leave blank to keep current secret'}
        />
      </div>
    </div>

    <div class="space-y-2">
      <label class="text-sm font-medium" for="connector-auto-workflow">Auto workflow</label>
      <Input
        id="connector-auto-workflow"
        value={draft.autoWorkflow}
        oninput={(event) => onDraftChange?.('autoWorkflow', event.currentTarget.value)}
        placeholder="triage"
      />
    </div>

    <div class="flex flex-wrap gap-2">
      <Button disabled={saving} onclick={onSave}>
        {editorMode === 'create' ? 'Create connector' : 'Save changes'}
      </Button>
      {#if editorMode === 'edit'}
        <Button variant="outline" onclick={onReset}>Reset editor</Button>
      {/if}
    </div>
  </Card.Content>
</Card.Root>
