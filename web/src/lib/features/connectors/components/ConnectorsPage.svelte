<script lang="ts">
  import { Waypoints } from '@lucide/svelte'
  import MasterDetailLayout from '$lib/components/layout/MasterDetailLayout.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import { Badge } from '$lib/components/ui/badge'
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

<div class="space-y-4">
  <MasterDetailLayout detailWidthClass="xl:grid-cols-[22rem_minmax(0,1fr)]">
    {#snippet main()}
      <ConnectorFleetPanel {controller} />
    {/snippet}

    {#snippet detail()}
      <SurfacePanel>
        {#snippet header()}
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2 text-sm font-semibold">
                <Waypoints class="size-4" />
                <span>{controller.selectedConnector()?.name || 'Create connector'}</span>
              </div>
              <p class="text-muted-foreground mt-1 text-sm leading-6">
                Keep sync wiring, health, and manual triggers inside the settings surface.
              </p>
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
        {/snippet}

        <div class="grid gap-4 p-4 xl:grid-cols-[minmax(0,1.25fr)_19rem]">
          <ConnectorFormPanel {controller} />
          <ConnectorStatusSidebar {controller} />
        </div>
      </SurfacePanel>

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
    {/snippet}
  </MasterDetailLayout>
</div>
