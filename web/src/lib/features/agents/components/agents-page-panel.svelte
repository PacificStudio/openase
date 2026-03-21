<script lang="ts">
  import { Button } from '$ui/button'
  import * as Tabs from '$ui/tabs'
  import { Plus } from '@lucide/svelte'
  import type { AgentInstance, ProviderConfig } from '../types'
  import AgentList from './agent-list.svelte'
  import ProviderList from './provider-list.svelte'

  let {
    activeTab = $bindable('instances'),
    agents,
    providers,
    loading = false,
    error = '',
    pageFeedback = '',
    canRegister = false,
    registerButtonTitle,
    onOpenRegister,
    onSelectTicket,
    onViewOutput,
    onConfigureProvider,
  }: {
    activeTab?: string
    agents: AgentInstance[]
    providers: ProviderConfig[]
    loading?: boolean
    error?: string
    pageFeedback?: string
    canRegister?: boolean
    registerButtonTitle?: string
    onOpenRegister?: () => void
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
    onConfigureProvider?: (provider: ProviderConfig) => void
  } = $props()
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-foreground text-lg font-semibold">Agents</h1>
    <Button
      size="sm"
      onclick={() => onOpenRegister?.()}
      disabled={!canRegister}
      title={registerButtonTitle}
    >
      <Plus class="size-3.5" />
      Register Agent
    </Button>
  </div>

  {#if loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-md border px-4 py-10 text-center text-sm"
    >
      Loading agents...
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    {#if pageFeedback}
      <div
        class="rounded-md border border-emerald-500/40 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-300"
      >
        {pageFeedback}
      </div>
    {/if}

    <Tabs.Root bind:value={activeTab}>
      <Tabs.List variant="line">
        <Tabs.Trigger value="instances">Instances</Tabs.Trigger>
        <Tabs.Trigger value="providers">Providers</Tabs.Trigger>
      </Tabs.List>
      <Tabs.Content value="instances" class="pt-3">
        <AgentList
          {agents}
          onSelectTicket={(ticketId) => onSelectTicket?.(ticketId)}
          onViewOutput={(agentId) => onViewOutput?.(agentId)}
        />
      </Tabs.Content>
      <Tabs.Content value="providers" class="pt-3">
        <ProviderList {providers} onConfigure={onConfigureProvider} />
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</div>
