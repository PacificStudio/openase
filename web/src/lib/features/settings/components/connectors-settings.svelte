<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import * as Card from '$ui/card'
  import { Separator } from '$ui/separator'
  import { ArrowDownToLine, Plug, RefreshCcw, Webhook } from '@lucide/svelte'

  const currentProjectName = $derived(appStore.currentProject?.name ?? 'this project')
  const inboundWebhookEndpoint = 'POST /api/v1/webhooks/:connector/:provider'
  const legacyGitHubEndpoint = 'POST /api/v1/webhooks/github'

  const runtimeSlices = [
    {
      title: 'Inbound webhook ingestion',
      description:
        'Providers can already push issue events into the runtime through the exported webhook receiver.',
      endpoint: inboundWebhookEndpoint,
      icon: Webhook,
    },
    {
      title: 'Legacy GitHub webhook compatibility',
      description:
        'The existing GitHub webhook path remains exported for ticket repo scope and PR lifecycle events.',
      endpoint: legacyGitHubEndpoint,
      icon: ArrowDownToLine,
    },
  ] as const

  const deferredActions = [
    'List project connectors',
    'Create or rotate connector credentials',
    'Pause, resume, test, or sync a connector on demand',
    'Inspect connector health, stats, or last error from Settings',
  ] as const
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Connectors</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Connector webhook ingestion is live. Project-scoped connector management is not yet available.
    </p>
  </div>

  <Separator />

  <div class="grid gap-6 xl:grid-cols-[minmax(0,1.1fr),minmax(0,0.9fr)]">
    <Card.Root>
      <Card.Header>
        <Card.Title>Current exported surface</Card.Title>
        <Card.Description>
          {currentProjectName} can receive connector traffic today, but the app only exposes the runtime
          ingress paths, not a full management plane.
        </Card.Description>
      </Card.Header>
      <Card.Content class="space-y-3">
        {#each runtimeSlices as slice (slice.title)}
          {@const Icon = slice.icon}
          <div class="border-border bg-muted/20 rounded-lg border p-4">
            <div class="flex items-start gap-3">
              <div class="bg-muted flex size-8 shrink-0 items-center justify-center rounded-md">
                <Icon class="text-muted-foreground size-4" />
              </div>
              <div class="min-w-0 space-y-2">
                <div class="text-sm font-medium">{slice.title}</div>
                <p class="text-muted-foreground text-sm">{slice.description}</p>
                <code class="bg-muted inline-flex rounded px-2 py-1 text-xs">{slice.endpoint}</code>
              </div>
            </div>
          </div>
        {/each}

        <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
          <div class="flex items-center gap-2 font-medium">
            <Plug class="text-muted-foreground size-4" />
            Backend runtime exists beyond the placeholder
          </div>
          <p class="text-muted-foreground mt-2">
            The repository already contains the issue connector domain model, registry, GitHub
            adapter, and connector sync orchestration. What is still missing is the project-facing
            CRUD and operator control boundary for Settings.
          </p>
        </div>
      </Card.Content>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Deferred management scope</Card.Title>
        <Card.Description>
          This section now documents the current boundary explicitly instead of showing a generic
          placeholder.
        </Card.Description>
      </Card.Header>
      <Card.Content class="space-y-4">
        <div class="border-border bg-muted/20 rounded-lg border p-4">
          <div class="flex items-center gap-2 text-sm font-medium">
            <RefreshCcw class="text-muted-foreground size-4" />
            Not exported yet
          </div>
          <ul class="text-muted-foreground mt-3 space-y-2 pl-5 text-sm">
            {#each deferredActions as action (action)}
              <li class="list-disc">{action}</li>
            {/each}
          </ul>
        </div>

        <div class="border-border bg-muted/20 rounded-lg border p-4 text-sm">
          <div class="font-medium">Next stable app-facing boundary</div>
          <p class="text-muted-foreground mt-2">
            When connector settings graduate from deferred scope, Settings should wire against
            dedicated project and connector endpoints for lifecycle, health checks, and manual sync
            control rather than inventing client-only state.
          </p>
        </div>
      </Card.Content>
    </Card.Root>
  </div>
</div>
