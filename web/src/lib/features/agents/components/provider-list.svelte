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
    {@const Icon = adapterIcons[provider.adapterType] ?? Bot}
    <Card.Root class="transition-colors hover:border-border/80">
      <Card.Header class="flex-row items-start justify-between gap-3 space-y-0 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="flex size-8 items-center justify-center rounded-md bg-muted">
            <Icon class="size-4 text-muted-foreground" />
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
          <span class="font-mono text-foreground">{provider.modelName}</span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Agents</span>
          <span class="tabular-nums text-foreground">{provider.agentCount}</span>
        </div>
      </Card.Content>
      <Card.Footer class="pt-2">
        <Button
          variant="outline"
          size="sm"
          class="w-full"
          disabled
          title="Provider editing is not wired in this slice"
        >
          <Settings class="size-3.5" />
          Configure
        </Button>
      </Card.Footer>
    </Card.Root>
  {/each}
</div>
