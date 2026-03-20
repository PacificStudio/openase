<script lang="ts">
  import { Cable } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import type { AgentProvider } from '../types'

  let {
    providers = [],
    provider = null,
    busy = false,
    error = '',
    countAgentsForProvider,
  }: {
    providers?: AgentProvider[]
    provider?: AgentProvider | null
    busy?: boolean
    error?: string
    countAgentsForProvider: (providerId: string) => number
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Cable class="size-4" />
      <span>Provider config sections</span>
    </CardTitle>
    <CardDescription>
      Provider configuration is loaded inside the agents feature instead of being reconstructed in
      the route.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-3">
    {#if busy}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border px-4 py-5 text-sm"
      >
        Loading provider config…
      </div>
    {:else if error}
      <div
        class="text-destructive border-destructive/25 bg-destructive/10 rounded-3xl border px-4 py-5 text-sm"
      >
        {error}
      </div>
    {:else if providers.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        No providers configured for the selected organization.
      </div>
    {:else}
      {#if provider}
        <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-sm font-semibold">{provider.name}</p>
            <Badge variant="secondary">{provider.adapter_type}</Badge>
            <Badge variant="outline">{countAgentsForProvider(provider.id)} agents</Badge>
          </div>
          <div class="mt-4 grid gap-3 sm:grid-cols-2">
            <div>
              <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">CLI</p>
              <p class="mt-2 text-sm font-semibold">
                {provider.cli_command}
                {provider.cli_args.join(' ')}
              </p>
            </div>
            <div>
              <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Model</p>
              <p class="mt-2 text-sm font-semibold">
                {provider.model_name || 'Default'} · {provider.model_max_tokens || 0} max tokens
              </p>
            </div>
          </div>
        </div>
      {/if}

      <ScrollPane class="max-h-64 space-y-3">
        {#each providers as item}
          <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <p class="text-sm font-semibold">{item.name}</p>
                  <Badge variant={provider?.id === item.id ? 'secondary' : 'outline'}>
                    {item.adapter_type}
                  </Badge>
                </div>
                <p class="text-muted-foreground mt-2 text-sm">
                  {item.model_name || 'Default model'} · temperature {item.model_temperature}
                </p>
              </div>
              <Badge variant="outline">{countAgentsForProvider(item.id)} agents</Badge>
            </div>
          </div>
        {/each}
      </ScrollPane>
    {/if}
  </CardContent>
</Card>
