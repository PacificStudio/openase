<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import type { ProjectCreationDraft } from '$lib/features/catalog-creation/model'
  import { projectStatusOptions } from '$lib/features/catalog-creation/model'
  import { adapterIconPath, providerAvailabilityLabel } from '$lib/features/providers'
  import { providerIsDispatchReady } from '$lib/features/providers'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import * as Collapsible from '$ui/collapsible'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { ChevronRight, Wrench } from '@lucide/svelte'

  let {
    draft,
    providers,
    creating = false,
    onNameInput,
    onSlugInput,
    onFieldChange,
    onSubmit,
  }: {
    draft: ProjectCreationDraft
    providers: AgentProvider[]
    creating?: boolean
    onNameInput?: (value: string) => void
    onSlugInput?: (value: string) => void
    onFieldChange?: (field: keyof ProjectCreationDraft, value: string) => void
    onSubmit?: () => void
  } = $props()

  let advancedOpen = $state(false)

  function selectedProvider() {
    return providers.find((item) => item.id === draft.defaultAgentProviderId) ?? null
  }
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Create project</Card.Title>
  </Card.Header>

  <Card.Content>
    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        onSubmit?.()
      }}
    >
      <div class="space-y-2">
        <Label for="project-name">Name</Label>
        <Input
          id="project-name"
          value={draft.name}
          placeholder="Automation Platform"
          oninput={(event) => onNameInput?.((event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="project-description">Description</Label>
        <Textarea
          id="project-description"
          rows={2}
          value={draft.description}
          placeholder="Brief project description"
          oninput={(event) =>
            onFieldChange?.('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.status}
            onValueChange={(value) => onFieldChange?.('status', value || 'Planned')}
          >
            <Select.Trigger class="w-full">{draft.status}</Select.Trigger>
            <Select.Content>
              {#each projectStatusOptions as status (status)}
                <Select.Item value={status}>{status}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Provider</Label>
          <Select.Root
            type="single"
            value={draft.defaultAgentProviderId}
            onValueChange={(value) => onFieldChange?.('defaultAgentProviderId', value || '')}
          >
            <Select.Trigger class="w-full">
              {@const provider = selectedProvider()}
              {#if provider}
                {@const iconPath = adapterIconPath(provider.adapter_type)}
                <span class="flex items-center gap-2 truncate">
                  {#if iconPath}
                    <img src={iconPath} alt="" class="size-4 shrink-0" />
                  {:else}
                    <Wrench class="text-muted-foreground size-4 shrink-0" />
                  {/if}
                  <span class="truncate">{provider.name}</span>
                  <span class="text-muted-foreground shrink-0 text-xs">
                    {providerAvailabilityLabel(provider.availability_state)}
                  </span>
                </span>
              {:else}
                None
              {/if}
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="">None</Select.Item>
              {#each providers as provider (provider.id)}
                {@const iconPath = adapterIconPath(provider.adapter_type)}
                <Select.Item value={provider.id}>
                  <span class="flex items-center gap-2">
                    {#if iconPath}
                      <img src={iconPath} alt="" class="size-4 shrink-0" />
                    {:else}
                      <Wrench class="text-muted-foreground size-4 shrink-0" />
                    {/if}
                    <span class="truncate">{provider.name}</span>
                    <span
                      class="shrink-0 text-xs {providerIsDispatchReady(provider.availability_state)
                        ? 'text-emerald-600'
                        : 'text-muted-foreground'}"
                    >
                      {providerAvailabilityLabel(provider.availability_state)}
                    </span>
                  </span>
                </Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      <Collapsible.Root bind:open={advancedOpen}>
        <Collapsible.Trigger>
          {#snippet child({ props })}
            <button
              {...props}
              type="button"
              class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm transition-colors"
            >
              <ChevronRight class="size-4 transition-transform {advancedOpen ? 'rotate-90' : ''}" />
              Advanced settings
            </button>
          {/snippet}
        </Collapsible.Trigger>
        <Collapsible.Content>
          <div class="mt-3 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="project-slug">Slug</Label>
              <Input
                id="project-slug"
                value={draft.slug}
                placeholder="auto-generated"
                oninput={(event) => onSlugInput?.((event.currentTarget as HTMLInputElement).value)}
              />
            </div>

            <div class="space-y-2">
              <Label for="project-max-concurrent-agents">Max agents</Label>
              <Input
                id="project-max-concurrent-agents"
                type="number"
                min="1"
                step="1"
                value={draft.maxConcurrentAgents}
                placeholder="Unlimited"
                oninput={(event) =>
                  onFieldChange?.(
                    'maxConcurrentAgents',
                    (event.currentTarget as HTMLInputElement).value,
                  )}
              />
            </div>
          </div>
        </Collapsible.Content>
      </Collapsible.Root>

      <div class="flex justify-end">
        <Button type="submit" disabled={creating}>
          {creating ? 'Creating…' : 'Create project'}
        </Button>
      </div>
    </form>
  </Card.Content>
</Card.Root>
