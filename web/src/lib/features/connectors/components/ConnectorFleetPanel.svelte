<script lang="ts">
  import { Cable, Plus } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import type { createConnectorsController } from '../controller.svelte'
  import {
    connectorSummary,
    countConnectors,
    formatTimestamp,
    statusBadgeVariant,
  } from '../presentation'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createConnectorsController>
  } = $props()
</script>

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
      <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Active</p>
        <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">
          {countConnectors(controller.connectors, 'active')}
        </p>
      </div>
      <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Paused</p>
        <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">
          {countConnectors(controller.connectors, 'paused')}
        </p>
      </div>
      <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Errors</p>
        <p class="mt-3 text-3xl font-semibold tracking-[-0.05em]">
          {countConnectors(controller.connectors, 'error')}
        </p>
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
                  {connector.type} · {connectorSummary(connector)}
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
