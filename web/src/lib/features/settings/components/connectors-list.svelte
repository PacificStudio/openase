<script lang="ts">
  import type { IssueConnectorRecord } from '$lib/api/openase'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'

  type ConnectorStats = {
    connector_id: string
    status: string
    last_sync_at?: string | null
    last_error: string
    stats: {
      total_synced: number
      synced24h: number
      failed_count: number
    }
  }

  let {
    projectName = 'this project',
    loading = false,
    connectors = [],
    busyConnectorId = '',
    connectorStats,
    onCreate,
    onEdit,
    onRefreshStats,
    onTest,
    onSync,
    onToggleStatus,
    onDelete,
  }: {
    projectName?: string
    loading?: boolean
    connectors?: IssueConnectorRecord[]
    busyConnectorId?: string
    connectorStats: (connector: IssueConnectorRecord) => ConnectorStats
    onCreate?: () => void
    onEdit?: (connector: IssueConnectorRecord) => void
    onRefreshStats?: (connectorId: string) => void
    onTest?: (connector: IssueConnectorRecord) => void
    onSync?: (connector: IssueConnectorRecord) => void
    onToggleStatus?: (connector: IssueConnectorRecord) => void
    onDelete?: (connector: IssueConnectorRecord) => void
  } = $props()

  function formatTimestamp(raw?: string | null) {
    if (!raw) return 'Never'

    const parsed = new Date(raw)
    if (Number.isNaN(parsed.getTime())) {
      return raw
    }

    return parsed.toLocaleString()
  }
</script>

<Card.Root>
  <Card.Header>
    <div class="flex items-center justify-between gap-3">
      <div>
        <Card.Title>Project connectors</Card.Title>
        <Card.Description>
          Manage the live connector surface for {projectName}, including runtime test, sync, status,
          and recent failure context.
        </Card.Description>
      </div>
      <Button variant="outline" onclick={onCreate}>New connector</Button>
    </div>
  </Card.Header>
  <Card.Content class="space-y-4">
    {#if loading}
      <div class="text-muted-foreground rounded-lg border border-dashed p-4 text-sm">
        Loading connectors...
      </div>
    {:else if connectors.length === 0}
      <div class="text-muted-foreground rounded-lg border border-dashed p-4 text-sm">
        No connectors configured yet. Create a GitHub issue connector to enable project-scoped pull
        sync, connection tests, and operator controls from Settings.
      </div>
    {:else}
      {#each connectors as connector (connector.id)}
        {@const stats = connectorStats(connector)}
        <div class="border-border rounded-xl border p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <div class="text-sm font-medium">{connector.name}</div>
                <span class="bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-[11px]">
                  {connector.type}
                </span>
                <span class="bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-[11px]">
                  {stats.status}
                </span>
              </div>
              <div class="text-muted-foreground text-sm">{connector.config.project_ref}</div>
            </div>

            <div class="flex flex-wrap gap-2">
              <Button variant="outline" onclick={() => onEdit?.(connector)}>Edit</Button>
              <Button
                variant="outline"
                disabled={busyConnectorId === connector.id}
                onclick={() => onRefreshStats?.(connector.id)}
              >
                Refresh stats
              </Button>
              <Button
                variant="outline"
                disabled={busyConnectorId === connector.id}
                onclick={() => onTest?.(connector)}
              >
                Test
              </Button>
              <Button
                variant="outline"
                disabled={busyConnectorId === connector.id}
                onclick={() => onSync?.(connector)}
              >
                Sync now
              </Button>
              <Button
                variant="outline"
                disabled={busyConnectorId === connector.id}
                onclick={() => onToggleStatus?.(connector)}
              >
                {connector.status === 'paused' ? 'Resume' : 'Pause'}
              </Button>
              <Button
                variant="outline"
                disabled={busyConnectorId === connector.id}
                onclick={() => onDelete?.(connector)}
              >
                Delete
              </Button>
            </div>
          </div>

          <div class="mt-4 grid gap-3 text-sm md:grid-cols-2">
            <div class="rounded-lg border border-dashed p-3">
              <div class="font-medium">Runtime policy</div>
              <div class="text-muted-foreground mt-2 space-y-1">
                <div>Direction: {connector.config.sync_direction}</div>
                <div>Poll interval: {connector.config.poll_interval}</div>
                <div>Labels: {connector.config.filters.labels.join(', ') || 'No filter'}</div>
              </div>
            </div>

            <div class="rounded-lg border border-dashed p-3">
              <div class="font-medium">Health and sync</div>
              <div class="text-muted-foreground mt-2 space-y-1">
                <div>Last sync: {formatTimestamp(stats.last_sync_at)}</div>
                <div>Total synced: {stats.stats.total_synced}</div>
                <div>Last 24h: {stats.stats.synced24h}</div>
                <div>Failures: {stats.stats.failed_count}</div>
              </div>
            </div>
          </div>

          <div class="mt-3 rounded-lg border border-dashed p-3 text-sm">
            <div class="font-medium">Last error</div>
            <div class="text-muted-foreground mt-2">
              {stats.last_error || 'No connector runtime errors recorded.'}
            </div>
          </div>
        </div>
      {/each}
    {/if}
  </Card.Content>
</Card.Root>
