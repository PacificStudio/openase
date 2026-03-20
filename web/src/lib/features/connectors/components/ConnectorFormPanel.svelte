<script lang="ts">
  import { Plus, Save, Trash2 } from '@lucide/svelte'
  import { Button } from '$lib/components/ui/button'
  import { inputClass, textAreaClass } from '$lib/features/workspace'
  import type { createConnectorsController } from '../controller.svelte'
  import { syncDirectionLabel } from '../presentation'
  import {
    connectorStatuses,
    connectorTypes,
    syncDirections,
    type ConnectorStatus,
    type ConnectorType,
    type SyncDirection,
  } from '../types'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createConnectorsController>
  } = $props()
</script>

<div class="space-y-5">
  <div class="grid gap-4 md:grid-cols-2">
    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Type
      </span>
      <select
        class={inputClass}
        value={controller.form.type}
        onchange={(event) =>
          controller.updateForm(
            'type',
            (event.currentTarget as HTMLSelectElement).value as ConnectorType,
          )}
      >
        {#each connectorTypes as connectorType}
          <option value={connectorType}>{connectorType}</option>
        {/each}
      </select>
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Status
      </span>
      <select
        class={inputClass}
        value={controller.form.status}
        onchange={(event) =>
          controller.updateForm(
            'status',
            (event.currentTarget as HTMLSelectElement).value as ConnectorStatus,
          )}
      >
        {#each connectorStatuses as connectorStatus}
          <option value={connectorStatus}>{connectorStatus}</option>
        {/each}
      </select>
    </label>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <label class="space-y-2 md:col-span-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Connector name
      </span>
      <input
        class={inputClass}
        value={controller.form.name}
        placeholder="GitHub · acme/backend"
        oninput={(event) =>
          controller.updateForm('name', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Base URL
      </span>
      <input
        class={inputClass}
        value={controller.form.base_url}
        placeholder="https://api.github.com"
        oninput={(event) =>
          controller.updateForm('base_url', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Project ref
      </span>
      <input
        class={inputClass}
        value={controller.form.project_ref}
        placeholder="acme/backend"
        oninput={(event) =>
          controller.updateForm('project_ref', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Auth token
      </span>
      <input
        class={inputClass}
        type="password"
        value={controller.form.auth_token}
        placeholder="ghp_xxx"
        oninput={(event) =>
          controller.updateForm('auth_token', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Webhook secret
      </span>
      <input
        class={inputClass}
        type="password"
        value={controller.form.webhook_secret}
        placeholder="optional secret"
        oninput={(event) =>
          controller.updateForm('webhook_secret', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>
  </div>

  <div class="grid gap-4 md:grid-cols-3">
    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Poll interval
      </span>
      <input
        class={inputClass}
        value={controller.form.poll_interval}
        placeholder="5m"
        oninput={(event) =>
          controller.updateForm('poll_interval', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Sync direction
      </span>
      <select
        class={inputClass}
        value={controller.form.sync_direction}
        onchange={(event) =>
          controller.updateForm(
            'sync_direction',
            (event.currentTarget as HTMLSelectElement).value as SyncDirection,
          )}
      >
        {#each syncDirections as direction}
          <option value={direction}>{syncDirectionLabel(direction)}</option>
        {/each}
      </select>
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Auto workflow
      </span>
      <input
        class={inputClass}
        value={controller.form.auto_workflow}
        placeholder="coding-default"
        oninput={(event) =>
          controller.updateForm('auto_workflow', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Include labels
      </span>
      <input
        class={inputClass}
        value={controller.form.labels}
        placeholder="openase, triaged"
        oninput={(event) =>
          controller.updateForm('labels', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Exclude labels
      </span>
      <input
        class={inputClass}
        value={controller.form.exclude_labels}
        placeholder="ignore-bot"
        oninput={(event) =>
          controller.updateForm('exclude_labels', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        States
      </span>
      <input
        class={inputClass}
        value={controller.form.states}
        placeholder="open, closed"
        oninput={(event) =>
          controller.updateForm('states', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>

    <label class="space-y-2">
      <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
        Authors
      </span>
      <input
        class={inputClass}
        value={controller.form.authors}
        placeholder="octocat, maintainer-bot"
        oninput={(event) =>
          controller.updateForm('authors', (event.currentTarget as HTMLInputElement).value)}
      />
    </label>
  </div>

  <label class="space-y-2">
    <span class="text-muted-foreground text-xs font-medium tracking-[0.22em] uppercase">
      Status mapping
    </span>
    <textarea
      class={textAreaClass}
      value={controller.form.status_mapping}
      placeholder={`open=Todo\nclosed=Done`}
      oninput={(event) =>
        controller.updateForm('status_mapping', (event.currentTarget as HTMLTextAreaElement).value)}
    ></textarea>
  </label>

  <div class="flex flex-wrap gap-3">
    <Button
      class="rounded-2xl"
      onclick={() => controller.saveCurrent()}
      disabled={controller.pendingAction === 'save'}
    >
      <Save class="size-4" />
      {controller.selectedConnector() ? 'Save connector' : 'Create connector'}
    </Button>
    <Button variant="outline" class="rounded-2xl" onclick={() => controller.startCreate()}>
      <Plus class="size-4" />
      New draft
    </Button>
    {#if controller.selectedConnector()}
      <Button
        variant="destructive"
        class="rounded-2xl"
        onclick={() => controller.removeCurrent()}
        disabled={controller.pendingAction === 'delete'}
      >
        <Trash2 class="size-4" />
        Remove
      </Button>
    {/if}
  </div>
</div>
