<script lang="ts">
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
    type CapabilityState,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import { Separator } from '$ui/separator'
  import * as Card from '$ui/card'

  type ConnectorBoundaryItem = {
    label: string
    location: string
    state: CapabilityState
    summary: string
  }

  const connectorsCapability = capabilityCatalog.connectorsSettings

  const boundaryItems: ConnectorBoundaryItem[] = [
    {
      label: 'Inbound webhook ingestion',
      location: 'Runtime / HTTP API',
      state: 'available',
      summary:
        'Current main accepts connector-driven inbound traffic through the runtime webhook receiver instead of a settings-managed connector inventory.',
    },
    {
      label: 'Project-scoped connector management',
      location: 'Settings / Connectors',
      state: 'deferred',
      summary:
        'List, create, update, pause, test, and sync controls are intentionally deferred until a stable app-facing management API is exported.',
    },
    {
      label: 'Connector sync operations',
      location: 'Runtime / Orchestrator',
      state: 'deferred',
      summary:
        'Connector domain models and sync orchestration exist in the backend, but they are not yet surfaced as stable operator controls in the app settings flow.',
    },
  ]

  const runtimeEndpoints = [
    {
      method: 'POST',
      path: '/api/v1/webhooks/github',
      summary: 'Legacy GitHub repo-scope webhook entrypoint.',
    },
    {
      method: 'POST',
      path: '/api/v1/webhooks/:connector/:provider',
      summary: 'Generic inbound webhook receiver for registered runtime endpoints.',
    },
  ] as const

  const deferredContract = [
    'GET /api/v1/projects/{projectId}/connectors',
    'POST /api/v1/projects/{projectId}/connectors',
    'PATCH /api/v1/connectors/{connectorId}',
    'POST /api/v1/connectors/{connectorId}/test',
    'POST /api/v1/connectors/{connectorId}/sync',
  ] as const

  const projectName = $derived(appStore.currentProject?.name ?? 'the active project')
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Connectors</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(connectorsCapability.state)}`}
      >
        {capabilityStateLabel(connectorsCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{connectorsCapability.summary}</p>
  </div>

  <Separator />

  <Card.Root>
    <Card.Header>
      <Card.Title>Current boundary</Card.Title>
      <Card.Description>
        Connector management for {projectName} is intentionally documented here instead of exposing incomplete
        settings controls.
      </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-3">
      {#each boundaryItems as item (item.label)}
        <div class="border-border rounded-md border px-3 py-3">
          <div class="flex flex-wrap items-center gap-2">
            <div class="text-foreground text-sm font-medium">{item.label}</div>
            <span
              class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(item.state)}`}
            >
              {capabilityStateLabel(item.state)}
            </span>
            <span class="text-muted-foreground text-xs">{item.location}</span>
          </div>
          <p class="text-muted-foreground mt-2 text-xs">{item.summary}</p>
        </div>
      {/each}
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Runtime surface on current main</Card.Title>
      <Card.Description>
        These routes are exported today and explain why Connectors are operationally present even
        though the settings slice is not yet editable.
      </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-3">
      {#each runtimeEndpoints as endpoint (endpoint.path)}
        <div class="border-border rounded-md border px-3 py-3">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span
              class="rounded border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 font-medium text-emerald-700 dark:text-emerald-300"
            >
              {endpoint.method}
            </span>
            <code class="bg-muted rounded px-2 py-0.5 text-xs">{endpoint.path}</code>
          </div>
          <p class="text-muted-foreground mt-2 text-xs">{endpoint.summary}</p>
        </div>
      {/each}
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Management contract still deferred</Card.Title>
      <Card.Description>
        The PRD already sketches the management endpoints below, but current main does not export
        them through the app settings path yet.
      </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-2">
      {#each deferredContract as endpoint (endpoint)}
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <code class="text-xs">{endpoint}</code>
        </div>
      {/each}
    </Card.Content>
  </Card.Root>
</div>
