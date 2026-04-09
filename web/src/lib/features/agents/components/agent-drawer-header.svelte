<script lang="ts">
  import { adapterIconPath } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import type { AgentProvider } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Check, Pencil, Wrench, X } from '@lucide/svelte'
  import { agentStatusDot, agentStatusLabel, agentStatusVariant } from './agent-drawer-state'
  import type { AgentInstance } from '../types'

  let {
    agent,
    providers = [],
    editingName = false,
    editNameValue = $bindable(''),
    savingName = false,
    savingProvider = false,
    onStartEditingName,
    onCancelEditingName,
    onSaveName,
    onNameKeydown,
    onProviderChange,
  }: {
    agent: AgentInstance
    providers?: AgentProvider[]
    editingName?: boolean
    editNameValue?: string
    savingName?: boolean
    savingProvider?: boolean
    onStartEditingName?: () => void
    onCancelEditingName?: () => void
    onSaveName?: () => void | Promise<void>
    onNameKeydown?: (event: KeyboardEvent) => void
    onProviderChange?: (providerId: string) => void | Promise<void>
  } = $props()

  const selectedProvider = $derived(
    providers.find((provider) => provider.id === agent.providerId) ?? null,
  )
</script>

<SheetHeader class="border-border border-b px-4 py-4 sm:px-6 sm:py-5">
  <div class="flex items-start gap-2 sm:gap-3">
    <span class={cn('mt-2 size-2.5 shrink-0 rounded-full', agentStatusDot[agent.status])}></span>
    <div class="min-w-0 flex-1">
      {#if editingName}
        <div class="flex items-center gap-1.5">
          <Input
            bind:value={editNameValue}
            class="h-8 text-base font-semibold"
            onkeydown={(event) => onNameKeydown?.(event)}
            disabled={savingName}
          />
          <Button
            variant="ghost"
            size="sm"
            class="size-7 shrink-0 p-0"
            disabled={savingName}
            aria-label="Save agent name"
            onclick={() => void onSaveName?.()}
          >
            <Check class="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="size-7 shrink-0 p-0"
            aria-label="Cancel agent rename"
            onclick={() => onCancelEditingName?.()}
          >
            <X class="size-3.5" />
          </Button>
        </div>
      {:else}
        <div class="group flex items-center gap-1.5">
          <SheetTitle class="truncate text-base">{agent.name}</SheetTitle>
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground opacity-0 transition-opacity group-hover:opacity-100"
            aria-label="Rename agent"
            onclick={() => onStartEditingName?.()}
            title="Rename agent"
          >
            <Pencil class="size-3" />
          </button>
        </div>
      {/if}
      <SheetDescription class="mt-1">
        <Select.Root
          type="single"
          value={agent.providerId}
          disabled={savingProvider || providers.length === 0}
          onValueChange={(value) => {
            if (value) void onProviderChange?.(value)
          }}
        >
          <Select.Trigger
            aria-label="Agent provider"
            class="text-muted-foreground hover:text-foreground h-auto w-auto gap-2 border-none bg-transparent p-0 text-[13px] shadow-none"
          >
            {#if selectedProvider}
              {@const iconPath = adapterIconPath(selectedProvider.adapter_type)}
              <div class="flex min-w-0 items-center gap-2">
                {#if iconPath}
                  <img src={iconPath} alt="" class="size-4 shrink-0" />
                {:else}
                  <Wrench class="text-muted-foreground size-4 shrink-0" />
                {/if}
                <span class="truncate">{selectedProvider.name} · {selectedProvider.model_name}</span
                >
              </div>
            {:else}
              {agent.providerName} · {agent.modelName}
            {/if}
          </Select.Trigger>
          <Select.Content align="start" class="min-w-56">
            {#each providers as provider (provider.id)}
              <Select.Item value={provider.id} label={provider.name}>
                {@const iconPath = adapterIconPath(provider.adapter_type)}
                <div class="flex min-w-0 items-center gap-2.5 py-0.5">
                  {#if iconPath}
                    <img src={iconPath} alt="" class="size-4 shrink-0" />
                  {:else}
                    <Wrench class="text-muted-foreground size-4 shrink-0" />
                  {/if}
                  <div class="min-w-0">
                    <div class="truncate text-sm">{provider.name}</div>
                    <div class="text-muted-foreground text-[11px]">
                      {provider.model_name} · {provider.adapter_type}
                    </div>
                  </div>
                </div>
              </Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </SheetDescription>
    </div>
    <span
      class={cn(
        'mt-0.5 inline-flex shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-medium',
        agentStatusVariant[agent.status],
      )}
    >
      {agentStatusLabel[agent.status]}
      {#if agent.runtimeControlState !== 'active'}
        <span class="ml-1 opacity-70">
          · {agent.runtimeControlState === 'pause_requested'
            ? 'Pausing'
            : agent.runtimeControlState === 'retired'
              ? 'Retired'
              : 'Paused'}
        </span>
      {/if}
    </span>
  </div>
</SheetHeader>
