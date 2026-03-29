<script lang="ts">
  import { Checkbox } from '$ui/checkbox'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import type { RepositoryDraft, RepositoryEditorMode } from '../repositories-model'

  let {
    mode,
    selectedRepo,
    draft,
    reposCount = 0,
    saving = false,
    onDraftChange,
  }: {
    mode: RepositoryEditorMode
    selectedRepo: ProjectRepoRecord | null
    draft: RepositoryDraft
    reposCount?: number
    saving?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
  } = $props()

  const primaryToggleLocked = $derived(
    (mode === 'create' && reposCount === 0) ||
      (mode === 'edit' && reposCount === 1 && Boolean(selectedRepo?.is_primary)),
  )
  const primaryHint = $derived.by(() => {
    if (mode === 'create' && reposCount === 0) {
      return 'The first repository is promoted to primary automatically.'
    }
    if (mode === 'edit' && reposCount === 1 && selectedRepo?.is_primary) {
      return 'A project with one repository always keeps that repository as primary.'
    }
    return 'The primary repository becomes the default repo scope for ticket-driven automation.'
  })

  function updateTextField(field: keyof RepositoryDraft, event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    onDraftChange?.(field, target.value)
  }
</script>

<div class="space-y-5">
  <section class="space-y-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Repository identity</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Basic Git coordinates that OpenASE uses for repo scopes and workspace preparation.
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="repo-name">Name</Label>
        <Input
          id="repo-name"
          value={draft.name}
          placeholder="backend"
          oninput={(event) => updateTextField('name', event)}
        />
      </div>

      <div class="space-y-2">
        <Label for="repo-default-branch">Default branch</Label>
        <Input
          id="repo-default-branch"
          value={draft.defaultBranch}
          placeholder="main"
          oninput={(event) => updateTextField('defaultBranch', event)}
        />
        <p class="text-muted-foreground text-xs">Blank input normalizes to `main`.</p>
      </div>
    </div>

    <div class="space-y-2">
      <Label for="repo-url">Repository URL</Label>
      <Input
        id="repo-url"
        value={draft.repositoryURL}
        placeholder="https://github.com/acme/backend.git"
        oninput={(event) => updateTextField('repositoryURL', event)}
      />
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Workspace mapping</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Optional checkout path and primary-repository routing behavior.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-workspace-dirname">Workspace dirname</Label>
      <Input
        id="repo-workspace-dirname"
        value={draft.workspaceDirname}
        placeholder="services/backend"
        oninput={(event) => updateTextField('workspaceDirname', event)}
      />
      <p class="text-muted-foreground text-xs">
        Leave empty to let the runtime derive the workspace path automatically.
      </p>
    </div>

    <div class="border-border rounded-xl border px-4 py-3">
      <div class="flex items-start gap-3">
        <Checkbox
          id="repo-primary"
          class="mt-0.5"
          checked={draft.isPrimary}
          aria-describedby="repo-primary-description"
          disabled={primaryToggleLocked || saving}
          onCheckedChange={(checked) => onDraftChange?.('isPrimary', checked)}
        />
        <div class="space-y-1">
          <Label for="repo-primary" class="text-foreground block text-sm font-medium">
            Primary repository
          </Label>
          <p id="repo-primary-description" class="text-muted-foreground text-xs">{primaryHint}</p>
        </div>
      </div>
    </div>
  </section>

  <section class="border-border space-y-4 border-t pt-6">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Metadata</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Labels help group repositories for workflows and ticket repo scopes.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="repo-labels">Labels</Label>
      <Textarea
        id="repo-labels"
        rows={4}
        value={draft.labels}
        placeholder={`go, backend, api\nworker`}
        oninput={(event) => updateTextField('labels', event)}
      />
      <p class="text-muted-foreground text-xs">
        Separate labels with commas or new lines. Empty entries are removed and duplicates are
        collapsed.
      </p>
    </div>
  </section>
</div>
