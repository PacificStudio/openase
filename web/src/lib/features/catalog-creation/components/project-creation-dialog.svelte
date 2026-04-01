<script lang="ts">
  import { goto } from '$app/navigation'
  import type { AgentProvider } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createProject } from '$lib/api/openase'
  import {
    createProjectDraft,
    parseProjectDraft,
    projectStatusOptions,
    slugFromName,
    type ProjectCreationDraft,
  } from '$lib/features/catalog-creation/model'
  import {
    adapterIconPath,
    providerAvailabilityLabel,
    providerIsDispatchReady,
  } from '$lib/features/providers'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Collapsible from '$ui/collapsible'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { ChevronRight, Wrench } from '@lucide/svelte'

  let {
    orgId,
    defaultProviderId = null,
    providers,
    open = $bindable(false),
  }: {
    orgId: string
    defaultProviderId?: string | null
    providers: AgentProvider[]
    open?: boolean
  } = $props()

  let draft = $state<ProjectCreationDraft>(createProjectDraft())
  let slugDirty = $state(false)
  let creating = $state(false)
  let advancedOpen = $state(false)

  $effect(() => {
    if (!open) {
      draft = createProjectDraft(defaultProviderId)
    }
  })

  function selectedProvider() {
    return providers.find((item) => item.id === draft.defaultAgentProviderId) ?? null
  }

  function reset() {
    draft = createProjectDraft(defaultProviderId)
    slugDirty = false
    creating = false
    advancedOpen = false
  }

  function updateName(value: string) {
    draft = {
      ...draft,
      name: value,
      slug: slugDirty ? draft.slug : slugFromName(value),
    }
  }

  function updateSlug(value: string) {
    slugDirty = true
    draft = { ...draft, slug: value }
  }

  function updateField(field: keyof ProjectCreationDraft, value: string) {
    draft = { ...draft, [field]: value }
  }

  async function handleSubmit() {
    const parsed = parseProjectDraft(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    creating = true

    try {
      const payload = await createProject(orgId, parsed.value)
      open = false
      reset()
      await goto(projectPath(orgId, payload.project.id))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create project.',
      )
    } finally {
      creating = false
    }
  }
</script>

<Dialog.Root
  bind:open
  onOpenChange={(next) => {
    if (!next) reset()
  }}
>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>Create project</Dialog.Title>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleSubmit()
      }}
    >
      <div class="space-y-2">
        <Label for="project-name">Name</Label>
        <Input
          id="project-name"
          value={draft.name}
          placeholder="Automation Platform"
          oninput={(event) => updateName((event.currentTarget as HTMLInputElement).value)}
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
            updateField('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.status}
            onValueChange={(value) => updateField('status', value || 'Planned')}
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
            onValueChange={(value) => updateField('defaultAgentProviderId', value || '')}
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
          <div class="mt-3 grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label for="project-slug">Slug</Label>
              <Input
                id="project-slug"
                value={draft.slug}
                placeholder="auto-generated"
                oninput={(event) => updateSlug((event.currentTarget as HTMLInputElement).value)}
              />
            </div>

            <div class="space-y-2">
              <Label for="project-max-agents">Max agents</Label>
              <Input
                id="project-max-agents"
                type="number"
                min="1"
                step="1"
                value={draft.maxConcurrentAgents}
                placeholder="Unlimited"
                oninput={(event) =>
                  updateField(
                    'maxConcurrentAgents',
                    (event.currentTarget as HTMLInputElement).value,
                  )}
              />
            </div>
          </div>
        </Collapsible.Content>
      </Collapsible.Root>

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>Cancel</Button>
          {/snippet}
        </Dialog.Close>
        <Button type="submit" disabled={creating}>
          {creating ? 'Creating...' : 'Create project'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
