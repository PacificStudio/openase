<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Bot, Settings, Cpu } from '@lucide/svelte'
  import type { ProviderConfig } from '../types'

  let { providers }: { providers: ProviderConfig[] } = $props()

  const adapterIcons: Record<string, typeof Bot> = {
    claude: Bot,
    codex: Cpu,
  }
</script>

<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
  {#each providers as provider (provider.id)}
    <Card.Root class="hover:border-border/80 transition-colors">
      <Card.Header class="flex-row items-start justify-between gap-3 space-y-0 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="bg-muted flex size-8 items-center justify-center rounded-md">
            {#if adapterIcons[provider.adapterType]}
              <svelte:component
                this={adapterIcons[provider.adapterType]}
                class="text-muted-foreground size-4"
              />
            {:else}
              <Bot class="text-muted-foreground size-4" />
            {/if}
          </div>
          <div>
            <div class="flex items-center gap-2">
              <Card.Title class="text-sm">{provider.name}</Card.Title>
              {#if provider.isDefault}
                <Badge variant="outline" class="text-[10px]">Default</Badge>
              {/if}
            </div>
            <Card.Description class="text-xs">{provider.adapterType}</Card.Description>
          </div>
        </div>
      </Card.Header>
      <Card.Content class="space-y-2 pt-0">
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Model</span>
          <span class="text-foreground font-mono">{provider.modelName}</span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Agents</span>
          <span class="text-foreground tabular-nums">{provider.agentCount}</span>
        </div>
      </Card.Content>
      <Card.Footer class="pt-2">
        <Button variant="outline" size="sm" class="w-full">
          <Settings class="size-3.5" />
          Configure
        </Button>
      </Card.Footer>
    </Card.Root>
  {/each}
</div>
