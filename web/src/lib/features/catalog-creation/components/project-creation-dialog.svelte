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
  import { projectPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'

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
  let error = $state('')

  $effect(() => {
    if (!open) {
      draft = createProjectDraft(defaultProviderId)
    }
  })

  function providerLabel(provider: AgentProvider) {
    return provider.available ? provider.name : `${provider.name} (Unavailable)`
  }

  function selectedProviderLabel() {
    const provider = providers.find((item) => item.id === draft.defaultAgentProviderId)
    return provider ? providerLabel(provider) : 'None'
  }

  function reset() {
    draft = createProjectDraft(defaultProviderId)
    slugDirty = false
    creating = false
    error = ''
  }

  function updateName(value: string) {
    draft = {
      ...draft,
      name: value,
      slug: slugDirty ? draft.slug : slugFromName(value),
    }
    error = ''
  }

  function updateSlug(value: string) {
    slugDirty = true
    draft = { ...draft, slug: value }
    error = ''
  }

  function updateField(field: keyof ProjectCreationDraft, value: string) {
    draft = { ...draft, [field]: value }
    error = ''
  }

  async function handleSubmit() {
    const parsed = parseProjectDraft(draft)
    if (!parsed.ok) {
      error = parsed.error
      return
    }

    creating = true
    error = ''

    try {
      const payload = await createProject(orgId, parsed.value)
      open = false
      reset()
      await goto(projectPath(orgId, payload.project.id))
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to create project.'
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
      <Dialog.Description>
        Set up a new project scope with its route slug, lifecycle, and concurrency limits.
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleSubmit()
      }}
    >
      <div class="grid gap-4 sm:grid-cols-2">
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
          <Label for="project-slug">Slug</Label>
          <Input
            id="project-slug"
            value={draft.slug}
            placeholder="automation-platform"
            oninput={(event) => updateSlug((event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="project-description">Description</Label>
        <Textarea
          id="project-description"
          rows={3}
          value={draft.description}
          placeholder="What this project owns and what success looks like."
          oninput={(event) =>
            updateField('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-3">
        <div class="space-y-2">
          <Label>Lifecycle</Label>
          <Select.Root
            type="single"
            value={draft.status}
            onValueChange={(value) => updateField('status', value || 'planning')}
          >
            <Select.Trigger class="w-full capitalize">{draft.status}</Select.Trigger>
            <Select.Content>
              {#each projectStatusOptions as status (status)}
                <Select.Item value={status} class="capitalize">{status}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
          <p class="text-muted-foreground text-xs">
            New projects also start with default board statuses: Backlog, Todo, In Progress, In
            Review, Done, and Cancelled.
          </p>
        </div>

        <div class="space-y-2">
          <Label for="project-max-agents">Max agents</Label>
          <Input
            id="project-max-agents"
            type="number"
            min="1"
            step="1"
            value={draft.maxConcurrentAgents}
            placeholder="Default"
            oninput={(event) =>
              updateField('maxConcurrentAgents', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label>Provider</Label>
          <Select.Root
            type="single"
            value={draft.defaultAgentProviderId}
            onValueChange={(value) => updateField('defaultAgentProviderId', value || '')}
          >
            <Select.Trigger class="w-full">{selectedProviderLabel()}</Select.Trigger>
            <Select.Content>
              <Select.Item value="">None</Select.Item>
              {#each providers as provider (provider.id)}
                <Select.Item value={provider.id}>{providerLabel(provider)}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      {#if providers.some((provider) => !provider.available)}
        <p class="text-muted-foreground text-xs">
          Unavailable providers are built into the organization, but their CLI is not currently on
          this machine's `PATH`.
        </p>
      {/if}

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

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
