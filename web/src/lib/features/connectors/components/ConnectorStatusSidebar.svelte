<script lang="ts">
  import {
    Activity,
    CheckCheck,
    CircleAlert,
    Pause,
    RefreshCcw,
    TestTubeDiagonal,
  } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import type { createConnectorsController } from '../controller.svelte'
  import { formatTimestamp, statusBadgeVariant } from '../presentation'

  let {
    controller,
  }: {
    controller: ReturnType<typeof createConnectorsController>
  } = $props()
</script>

<div class="space-y-4">
  <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
    <div class="flex items-center gap-2 text-sm font-semibold">
      <Activity class="size-4" />
      <span>Status</span>
    </div>
    <div class="mt-4 space-y-3 text-sm">
      <div class="flex items-center justify-between gap-3">
        <span class="text-muted-foreground">Lifecycle</span>
        <Badge
          variant={statusBadgeVariant(
            controller.selectedConnector()?.status || controller.form.status,
          )}
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

  <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
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
        disabled={!controller.selectedConnectorId || controller.pendingAction === 'toggle-status'}
      >
        <Pause class="size-4" />
        {controller.selectedConnector()?.status === 'paused'
          ? 'Resume connector'
          : 'Pause connector'}
      </Button>
    </div>
  </div>

  <div class="border-border/70 bg-background/70 rounded-3xl border px-4 py-4">
    <div class="flex items-center gap-2 text-sm font-semibold">
      <CircleAlert class="size-4" />
      <span>Notes</span>
    </div>
    <div class="text-muted-foreground mt-4 space-y-3 text-sm leading-6">
      <p>Inbound webhook entrypoint: <code>/api/v1/connectors/inbound-webhook</code></p>
      <p>
        Use <code>state=status-name</code> mappings to bind external issue states to OpenASE ticket lanes.
      </p>
      <p>
        Local draft mode is a narrow vertical slice so the UI stays testable before the connector
        CRUD API is fully wired in the backend.
      </p>
    </div>
  </div>
</div>
