<script lang="ts">
  import {
    Activity,
    Cable,
    CheckCheck,
    CircleAlert,
    Pause,
    Plus,
    RefreshCcw,
    Save,
    TestTubeDiagonal,
    Trash2,
    Waypoints,
  } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { inputClass, textAreaClass } from '$lib/features/workspace'
  import type { createConnectorsController } from '../controller.svelte'
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

  function statusBadgeVariant(status: string): 'secondary' | 'destructive' | 'outline' {
    switch (status) {
      case 'active':
        return 'secondary'
      case 'error':
        return 'destructive'
      default:
        return 'outline'
    }
  }

  function syncDirectionLabel(direction: string) {
    switch (direction) {
      case 'pull_only':
        return 'Pull only'
      case 'push_only':
        return 'Push only'
      default:
        return 'Bidirectional'
    }
  }

  function formatTimestamp(value?: string | null) {
    if (!value) {
      return 'Never'
    }

    const parsed = new Date(value)
    return Number.isNaN(parsed.valueOf()) ? value : parsed.toLocaleString()
  }

  function connectorSummary(connectorId: string) {
    const connector = controller.connectors.find((item) => item.id === connectorId) ?? null
    if (!connector) {
      return ''
    }

    if (connector.type === 'inbound-webhook') {
      return 'Push ingress only'
    }

    return syncDirectionLabel(connector.config.sync_direction)
  }
</script>

<svelte:head>
  <title>Connectors · OpenASE</title>
</svelte:head>

<div class="space-y-6">
  <div class="grid gap-6 xl:grid-cols-[24rem_minmax(0,1fr)]">
    <div class="space-y-6">
      <Card class="border-border/80 bg-background/85 backdrop-blur">
        <CardHeader>
          <div class="flex items-center justify-between gap-3">
            <div>
              <CardTitle class="flex items-center gap-2">
                <Cable class="size-4" />
                <span>Connector fleet</span>
              </CardTitle>
              <CardDescription>
                List, inspect, and trigger sync entry points for the active project.
              </CardDescription>
            </div>
            <Button variant="outline" size="sm" onclick={() => controller.startCreate()}>
              <Plus class="size-4" />
              Add
            </Button>
          </div>
        </CardHeader>
        <CardContent class="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
          <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
            <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Active</p>
            <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">{controller.activeCount()}</p>
          </div>
          <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
            <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Paused</p>
            <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">{controller.pausedCount()}</p>
          </div>
          <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
            <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Errors</p>
            <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">{controller.errorCount()}</p>
          </div>
        </CardContent>
      </Card>

      <Card class="border-border/80 bg-background/85 backdrop-blur">
        <CardHeader>
          <CardTitle>Connector list</CardTitle>
          <CardDescription>
            {#if controller.persistenceMode === 'api'}
              Live project connectors from the HTTP API.
            {:else}
              Browser-local connector drafts while the HTTP API slice lands.
            {/if}
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          {#if controller.connectors.length === 0 && !controller.busy}
            <div
              class="text-muted-foreground border-border/70 bg-muted/35 rounded-3xl border border-dashed px-4 py-6 text-sm"
            >
              No connectors yet. Add one to seed the connector management surface.
            </div>
          {:else}
            {#each controller.connectors as connector}
              <button
                type="button"
                class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
                  controller.selectedConnectorId === connector.id
                    ? 'border-foreground/30 bg-foreground text-background shadow-lg shadow-black/10'
                    : 'border-border/70 bg-background/65 hover:border-foreground/15 hover:bg-background'
                }`}
                onclick={() => controller.selectConnector(connector.id)}
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold">{connector.name || 'Untitled connector'}</p>
                    <p
                      class={`mt-1 text-xs font-medium uppercase ${
                        controller.selectedConnectorId === connector.id
                          ? 'text-background/70'
                          : 'text-muted-foreground'
                      }`}
                    >
                      {connector.type} · {connectorSummary(connector.id)}
                    </p>
                  </div>
                  <Badge variant={statusBadgeVariant(connector.status)}>{connector.status}</Badge>
                </div>
                <div
                  class={`mt-4 grid gap-2 text-xs ${
                    controller.selectedConnectorId === connector.id
                      ? 'text-background/75'
                      : 'text-muted-foreground'
                  }`}
                >
                  <p>Project ref: {connector.config.project_ref || 'Not set'}</p>
                  <p>Last sync: {formatTimestamp(connector.last_sync_at)}</p>
                  <p>Total synced: {connector.stats.total_synced}</p>
                </div>
              </button>
            {/each}
          {/if}
        </CardContent>
      </Card>
    </div>

    <div class="space-y-6">
      <Card class="border-border/80 bg-background/85 backdrop-blur">
        <CardHeader>
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle class="flex items-center gap-2">
                <Waypoints class="size-4" />
                <span>{controller.selectedConnector()?.name || 'Create connector'}</span>
              </CardTitle>
              <CardDescription>
                Keep config, status view, and trigger actions in the same Phase 2 feature boundary.
              </CardDescription>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{controller.persistenceMode === 'api' ? 'API mode' : 'Local draft mode'}</Badge>
              {#if controller.selectedConnector()}
                <Badge variant={statusBadgeVariant(controller.selectedConnector()?.status || 'paused')}>
                  {controller.selectedConnector()?.status}
                </Badge>
              {/if}
            </div>
          </div>
        </CardHeader>
        <CardContent class="grid gap-6 xl:grid-cols-[minmax(0,1.25fr)_20rem]">
          <div class="space-y-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label class="space-y-2">
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
                  Webhook secret
                </span>
                <input
                  class={inputClass}
                  type="password"
                  value={controller.form.webhook_secret}
                  placeholder="optional secret"
                  oninput={(event) =>
                    controller.updateForm(
                      'webhook_secret',
                      (event.currentTarget as HTMLInputElement).value,
                    )}
                />
              </label>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
              <label class="space-y-2">
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
                  Poll interval
                </span>
                <input
                  class={inputClass}
                  value={controller.form.poll_interval}
                  placeholder="5m"
                  oninput={(event) =>
                    controller.updateForm(
                      'poll_interval',
                      (event.currentTarget as HTMLInputElement).value,
                    )}
                />
              </label>

              <label class="space-y-2">
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
                  Auto workflow
                </span>
                <input
                  class={inputClass}
                  value={controller.form.auto_workflow}
                  placeholder="coding-default"
                  oninput={(event) =>
                    controller.updateForm(
                      'auto_workflow',
                      (event.currentTarget as HTMLInputElement).value,
                    )}
                />
              </label>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <label class="space-y-2">
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
                  Exclude labels
                </span>
                <input
                  class={inputClass}
                  value={controller.form.exclude_labels}
                  placeholder="ignore-bot"
                  oninput={(event) =>
                    controller.updateForm(
                      'exclude_labels',
                      (event.currentTarget as HTMLInputElement).value,
                    )}
                />
              </label>

              <label class="space-y-2">
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
                <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
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
              <span class="text-xs font-medium tracking-[0.22em] uppercase text-muted-foreground">
                Status mapping
              </span>
              <textarea
                class={textAreaClass}
                value={controller.form.status_mapping}
                placeholder={`open=Todo\nclosed=Done`}
                oninput={(event) =>
                  controller.updateForm(
                    'status_mapping',
                    (event.currentTarget as HTMLTextAreaElement).value,
                  )}
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
              <Button
                variant="outline"
                class="rounded-2xl"
                onclick={() => controller.startCreate()}
              >
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

          <div class="space-y-4">
            <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
              <div class="flex items-center gap-2 text-sm font-semibold">
                <Activity class="size-4" />
                <span>Status</span>
              </div>
              <div class="mt-4 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">Lifecycle</span>
                  <Badge
                    variant={statusBadgeVariant(controller.selectedConnector()?.status || controller.form.status)}
                  >
                    {controller.selectedConnector()?.status || controller.form.status}
                  </Badge>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">Last sync</span>
                  <span>{formatTimestamp(controller.selectedConnector()?.last_sync_at)}</span>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">Synced issues</span>
                  <span>{controller.selectedConnector()?.stats.total_synced || 0}</span>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">24h window</span>
                  <span>{controller.selectedConnector()?.stats.synced_24h || 0}</span>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span class="text-muted-foreground">Failures</span>
                  <span>{controller.selectedConnector()?.stats.failed_count || 0}</span>
                </div>
              </div>
            </div>

            <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
              <div class="flex items-center gap-2 text-sm font-semibold">
                <CheckCheck class="size-4" />
                <span>Quick actions</span>
              </div>
              <div class="mt-4 grid gap-3">
                <Button
                  variant="outline"
                  class="justify-start rounded-2xl"
                  onclick={() => controller.runSync(controller.selectedConnectorId)}
                  disabled={!controller.selectedConnectorId || controller.pendingAction === 'sync'}
                >
                  <RefreshCcw class="size-4" />
                  Sync once
                </Button>
                <Button
                  variant="outline"
                  class="justify-start rounded-2xl"
                  onclick={() => controller.runTest(controller.selectedConnectorId)}
                  disabled={!controller.selectedConnectorId || controller.pendingAction === 'test'}
                >
                  <TestTubeDiagonal class="size-4" />
                  Test connection
                </Button>
                <Button
                  variant="outline"
                  class="justify-start rounded-2xl"
                  onclick={() => controller.toggleStatus(controller.selectedConnectorId)}
                  disabled={
                    !controller.selectedConnectorId || controller.pendingAction === 'toggle-status'
                  }
                >
                  <Pause class="size-4" />
                  {controller.selectedConnector()?.status === 'paused' ? 'Resume connector' : 'Pause connector'}
                </Button>
              </div>
            </div>

            <div class="rounded-3xl border border-border/70 bg-background/70 px-4 py-4">
              <div class="flex items-center gap-2 text-sm font-semibold">
                <CircleAlert class="size-4" />
                <span>Notes</span>
              </div>
              <div class="text-muted-foreground mt-4 space-y-3 text-sm leading-6">
                <p>Inbound webhook entrypoint: <code>/api/v1/connectors/inbound-webhook</code></p>
                <p>Use <code>state=status-name</code> mappings to bind external issue states to OpenASE ticket lanes.</p>
                <p>Local draft mode is a narrow vertical slice so the UI stays testable before the connector CRUD API is fully wired in the backend.</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {#if controller.notice}
        <div class="rounded-3xl border border-emerald-500/30 bg-emerald-500/10 px-5 py-4 text-sm">
          {controller.notice}
        </div>
      {/if}

      {#if controller.error}
        <div class="rounded-3xl border border-destructive/30 bg-destructive/10 px-5 py-4 text-sm text-destructive">
          {controller.error}
        </div>
      {/if}
    </div>
  </div>
</div>
