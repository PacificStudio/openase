<script lang="ts">
  import { Waypoints } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import type { createConnectorsController } from '../controller.svelte'
  import { statusBadgeVariant } from '../presentation'
  import ConnectorFleetPanel from './ConnectorFleetPanel.svelte'
  import ConnectorFormPanel from './ConnectorFormPanel.svelte'
  import ConnectorStatusSidebar from './ConnectorStatusSidebar.svelte'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createConnectorsController>
  } = $props()
</script>

<svelte:head>
  <title>Connectors · OpenASE</title>
</svelte:head>

<div class="space-y-6">
  <div class="grid gap-6 xl:grid-cols-[24rem_minmax(0,1fr)]">
    <ConnectorFleetPanel {controller} />

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
              <Badge variant="outline">
                {controller.persistenceMode === 'api' ? 'API mode' : 'Local draft mode'}
              </Badge>
              {#if controller.selectedConnector()}
                <Badge
                  variant={statusBadgeVariant(controller.selectedConnector()?.status || 'paused')}
                >
                  {controller.selectedConnector()?.status}
                </Badge>
              {/if}
            </div>
          </div>
        </CardHeader>
        <CardContent class="grid gap-6 xl:grid-cols-[minmax(0,1.25fr)_20rem]">
          <ConnectorFormPanel {controller} />
          <ConnectorStatusSidebar {controller} />
        </CardContent>
      </Card>

      {#if controller.notice}
        <div class="rounded-3xl border border-emerald-500/30 bg-emerald-500/10 px-5 py-4 text-sm">
          {controller.notice}
        </div>
      {/if}

      {#if controller.error}
        <div
          class="border-destructive/30 bg-destructive/10 text-destructive rounded-3xl border px-5 py-4 text-sm"
        >
          {controller.error}
        </div>
      {/if}
    </div>
  </div>
</div>
