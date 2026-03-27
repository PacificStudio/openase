<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { providerAvailabilityLabel, ProviderAvailabilityBadge } from '$lib/features/providers'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Bot, Cpu } from '@lucide/svelte'
  import type { ProviderOption } from './agent-settings-model'

  const inheritProviderValue = '__org_default__'
  const adapterIcons: Record<string, typeof Bot> = {
    claude: Bot,
    codex: Cpu,
  }

  let {
    providers,
    selectedDefaultProviderId,
    selectedDefaultProviderName = null,
    orgDefaultProviderId = null,
    orgDefaultProviderName = null,
    saving = false,
    onSelectionChange,
    onSave,
  }: {
    providers: ProviderOption[]
    selectedDefaultProviderId: string
    selectedDefaultProviderName?: string | null
    orgDefaultProviderId?: string | null
    orgDefaultProviderName?: string | null
    saving?: boolean
    onSelectionChange?: (value: string) => void
    onSave?: () => void
  } = $props()
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Defaults</Card.Title>
    <Card.Description>
      Set project-level routing defaults without duplicating runtime controls.
    </Card.Description>
  </Card.Header>
  <Card.Content class="space-y-4">
    <div class="space-y-2">
      <Label>Default agent provider</Label>
      <Select.Root
        type="single"
        value={selectedDefaultProviderId || inheritProviderValue}
        onValueChange={(value) => {
          onSelectionChange?.(value === inheritProviderValue ? '' : value || '')
        }}
      >
        <Select.Trigger class="w-full">
          {selectedDefaultProviderName ??
            (selectedDefaultProviderId ? 'Unavailable provider' : 'Use organization default')}
        </Select.Trigger>
        <Select.Content>
          <Select.Item value={inheritProviderValue}>
            Use organization default
            {#if orgDefaultProviderName}
              · {orgDefaultProviderName}
            {/if}
          </Select.Item>
          {#each providers as provider (provider.id)}
            <Select.Item value={provider.id}>
              {provider.name}
              {' '}· {provider.machineName}
              {' '}· {providerAvailabilityLabel(provider.availabilityState)}
              {' '}· {provider.adapterType} · {provider.modelName}
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div
      class="border-border bg-muted/20 text-muted-foreground rounded-md border px-3 py-3 text-xs"
    >
      Max concurrent agents remains in `Settings / General` and is currently set to
      {` ${appStore.currentProject?.max_concurrent_agents ?? 0}`}.
    </div>

    <div class="space-y-2">
      <div class="text-foreground text-sm font-medium">Providers in scope</div>
      {#if providers.length === 0}
        <div class="text-muted-foreground text-xs">
          No providers are registered for this organization yet.
        </div>
      {:else}
        <div class="space-y-2">
          {#each providers as provider (provider.id)}
            {@const Icon = adapterIcons[provider.adapterType] ?? Bot}
            <div
              class="border-border flex items-start justify-between gap-3 rounded-md border px-3 py-2"
            >
              <div class="flex items-start gap-2">
                <div class="bg-muted mt-0.5 flex size-7 items-center justify-center rounded-md">
                  <Icon class="text-muted-foreground size-3.5" />
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-foreground truncate text-sm font-medium">{provider.name}</span
                    >
                    <ProviderAvailabilityBadge
                      availabilityState={provider.availabilityState}
                      availabilityReason={provider.availabilityReason}
                      availabilityCheckedAt={provider.availabilityCheckedAt}
                      class="text-[10px]"
                    />
                    {#if selectedDefaultProviderId === provider.id}
                      <Badge variant="outline" class="text-[10px]">Project default</Badge>
                    {:else if orgDefaultProviderId === provider.id}
                      <Badge variant="secondary" class="text-[10px]">Org default</Badge>
                    {/if}
                  </div>
                  <div class="text-muted-foreground text-xs">
                    {provider.adapterType} · {provider.modelName}
                  </div>
                </div>
              </div>
              <div class="text-muted-foreground text-right text-xs">
                {provider.agentCount} agents
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </Card.Content>
  <Card.Footer>
    <Button onclick={onSave} disabled={saving || providers.length === 0}>
      {saving ? 'Saving…' : 'Save default provider'}
    </Button>
  </Card.Footer>
</Card.Root>
