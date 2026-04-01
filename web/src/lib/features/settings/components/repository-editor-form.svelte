<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { ChevronDown } from '@lucide/svelte'
  import type { RepositoryDraft } from '../repositories-model'

  let {
    draft,
    compact = false,
    onDraftChange,
  }: {
    draft: RepositoryDraft
    compact?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
  } = $props()

  let showAdvanced = $state(false)

  function update(field: keyof RepositoryDraft, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }
</script>

<div class="space-y-4" data-testid="repository-editor-form">
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for="repo-name" class="text-xs">Name</Label>
      <Input
        id="repo-name"
        value={draft.name}
        placeholder="backend"
        class="h-9 text-sm"
        oninput={(event) => update('name', event)}
      />
    </div>
    <div class="space-y-1.5">
      <Label for="repo-default-branch" class="text-xs">Default branch</Label>
      <Input
        id="repo-default-branch"
        value={draft.defaultBranch}
        placeholder="main"
        class="h-9 text-sm"
        oninput={(event) => update('defaultBranch', event)}
      />
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for="repo-url" class="text-xs">Repository URL</Label>
    <Input
      id="repo-url"
      value={draft.repositoryURL}
      placeholder="https://github.com/acme/backend.git"
      class="h-9 text-sm"
      oninput={(event) => update('repositoryURL', event)}
    />
  </div>

  {#if !compact}
    <div class="space-y-1.5">
      <Label for="repo-labels" class="text-xs">Labels</Label>
      <Textarea
        id="repo-labels"
        rows={2}
        value={draft.labels}
        placeholder="go, backend, api"
        class="text-sm"
        oninput={(event) => update('labels', event)}
      />
    </div>

    <div>
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground hover:text-foreground h-7 gap-1 px-0 text-xs"
        onclick={() => (showAdvanced = !showAdvanced)}
      >
        Advanced
        <ChevronDown class="size-3 transition-transform {showAdvanced ? 'rotate-180' : ''}" />
      </Button>

      {#if showAdvanced}
        <div class="mt-2 space-y-1.5">
          <Label for="repo-workspace-dirname" class="text-xs">Workspace path</Label>
          <Input
            id="repo-workspace-dirname"
            value={draft.workspaceDirname}
            placeholder="Auto-derived from repo name"
            class="h-9 text-sm"
            oninput={(event) => update('workspaceDirname', event)}
          />
          <p class="text-muted-foreground text-[11px]">
            Override the checkout subdirectory for monorepo setups. Leave empty for default.
          </p>
        </div>
      {/if}
    </div>
  {/if}
</div>
