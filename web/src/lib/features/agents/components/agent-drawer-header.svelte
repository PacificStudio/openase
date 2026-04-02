<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { adapterIconPath } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Check, Pencil, Wrench, X } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let {
    agent,
    providers = [],
    editingName = false,
    editNameValue = '',
    savingName = false,
    savingProvider = false,
    onStartEditingName,
    onCancelEditingName,
    onSaveName,
    onProviderChange,
    onEditNameValueChange,
    onNameKeydown,
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
    onProviderChange?: (providerId: string) => void | Promise<void>
    onEditNameValueChange?: (value: string) => void
    onNameKeydown?: (event: KeyboardEvent) => void
  } = $props()

  const statusVariant: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400',
    claimed: 'bg-amber-500/15 text-amber-700 dark:text-amber-400',
    running: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
    paused: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
    failed: 'bg-red-500/15 text-red-700 dark:text-red-400',
    terminated: 'bg-slate-500/15 text-slate-600 dark:text-slate-400',
  }

  const statusDot: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    claimed: 'bg-amber-500',
    running: 'bg-blue-500',
    paused: 'bg-orange-500',
    failed: 'bg-red-500',
    terminated: 'bg-slate-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    claimed: 'Claimed',
    running: 'Running',
    paused: 'Paused',
    failed: 'Failed',
    terminated: 'Terminated',
  }

  const selectedProvider = $derived(
    providers.find((provider) => provider.id === agent.providerId) ?? null,
  )
</script>

<SheetHeader class="border-border border-b px-6 py-5">
  <div class="flex items-start gap-3">
    <span class={cn('mt-2 size-2.5 shrink-0 rounded-full', statusDot[agent.status])}></span>
    <div class="min-w-0 flex-1">
      {#if editingName}
        <div class="flex items-center gap-1.5">
          <Input
            value={editNameValue}
            class="h-8 text-base font-semibold"
            oninput={(event) =>
              onEditNameValueChange?.((event.currentTarget as HTMLInputElement).value)}
            onkeydown={onNameKeydown}
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
            onclick={onCancelEditingName}
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
            onclick={onStartEditingName}
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
                <span class="truncate">{selectedProvider.name} · {selectedProvider.model_name}</span>
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
        statusVariant[agent.status],
      )}
    >
      {statusLabels[agent.status]}
      {#if agent.runtimeControlState !== 'active'}
        <span class="ml-1 opacity-70">
          · {agent.runtimeControlState === 'pause_requested' ? 'Pausing' : 'Paused'}
        </span>
      {/if}
    </span>
  </div>
</SheetHeader>
