<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import type { ProjectCreationDraft } from '$lib/features/catalog-creation/model'
  import { projectStatusOptions } from '$lib/features/catalog-creation/model'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'

  let {
    draft,
    providers,
    creating = false,
    error = '',
    onNameInput,
    onSlugInput,
    onFieldChange,
    onSubmit,
  }: {
    draft: ProjectCreationDraft
    providers: AgentProvider[]
    creating?: boolean
    error?: string
    onNameInput?: (value: string) => void
    onSlugInput?: (value: string) => void
    onFieldChange?: (field: keyof ProjectCreationDraft, value: string) => void
    onSubmit?: () => void
  } = $props()
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Create project</Card.Title>
    <Card.Description>
      Set the initial route slug, lifecycle status, and concurrency envelope for a new project.
    </Card.Description>
  </Card.Header>

  <Card.Content>
    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        onSubmit?.()
      }}
    >
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label for="project-name">Project name</Label>
          <Input
            id="project-name"
            value={draft.name}
            placeholder="Automation Platform"
            oninput={(event) => onNameInput?.((event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label for="project-slug">Slug</Label>
          <Input
            id="project-slug"
            value={draft.slug}
            placeholder="automation-platform"
            oninput={(event) => onSlugInput?.((event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="project-description">Description</Label>
        <Textarea
          id="project-description"
          rows={4}
          value={draft.description}
          placeholder="What this project owns, which operators it serves, and what success looks like."
          oninput={(event) =>
            onFieldChange?.('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.status}
            onValueChange={(value) => onFieldChange?.('status', value || 'planning')}
          >
            <Select.Trigger class="w-full capitalize">{draft.status}</Select.Trigger>
            <Select.Content>
              {#each projectStatusOptions as status (status)}
                <Select.Item value={status} class="capitalize">{status}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label for="project-max-concurrent-agents">Max concurrent agents</Label>
          <Input
            id="project-max-concurrent-agents"
            type="number"
            min="1"
            step="1"
            value={draft.maxConcurrentAgents}
            placeholder="Leave blank for backend default"
            oninput={(event) =>
              onFieldChange?.(
                'maxConcurrentAgents',
                (event.currentTarget as HTMLInputElement).value,
              )}
          />
        </div>

        <div class="space-y-2">
          <Label>Default provider</Label>
          <Select.Root
            type="single"
            value={draft.defaultAgentProviderId}
            onValueChange={(value) => onFieldChange?.('defaultAgentProviderId', value || '')}
          >
            <Select.Trigger class="w-full">
              {providers.find((provider) => provider.id === draft.defaultAgentProviderId)?.name ??
                'No default provider'}
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="">No default provider</Select.Item>
              {#each providers as provider (provider.id)}
                <Select.Item value={provider.id}>{provider.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

      <div class="flex items-center justify-between gap-3">
        <p class="text-muted-foreground text-xs">
          Accessible machines can stay empty on creation and be refined after the project is live.
        </p>
        <Button type="submit" disabled={creating}>
          {creating ? 'Creating…' : 'Create project'}
        </Button>
      </div>
    </form>
  </Card.Content>
</Card.Root>
