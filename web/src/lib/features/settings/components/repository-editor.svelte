<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import type { RepositoryDraft, RepositoryEditorMode } from '../repositories-model'

  let {
    mode,
    selectedRepo,
    draft,
    reposCount = 0,
    loading = false,
    saving = false,
    deleting = false,
    onDraftChange,
    onSave,
    onDelete,
    onReset,
  }: {
    mode: RepositoryEditorMode
    selectedRepo: ProjectRepoRecord | null
    draft: RepositoryDraft
    reposCount?: number
    loading?: boolean
    saving?: boolean
    deleting?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
    onSave?: () => void
    onDelete?: () => void
    onReset?: () => void
  } = $props()

  const canDelete = $derived(mode === 'edit' && Boolean(selectedRepo))
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

<div class="border-border bg-card rounded-2xl border">
  <div class="border-border flex flex-wrap items-start justify-between gap-3 border-b px-5 py-4">
    <div>
      <h3 class="text-foreground text-base font-semibold">
        {mode === 'create' ? 'Add repository' : (selectedRepo?.name ?? 'Edit repository')}
      </h3>
      <p class="text-muted-foreground mt-1 max-w-2xl text-sm">
        Configure the Git repository metadata that ticket repo scopes, workflows, and workspace
        bootstrap logic consume.
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <Button variant="outline" onclick={onReset} disabled={saving || deleting || loading}>
        Reset
      </Button>
      <Button onclick={onSave} disabled={saving || deleting || loading}>
        {saving ? 'Saving…' : mode === 'create' ? 'Create repository' : 'Save changes'}
      </Button>
      <Button variant="destructive" onclick={onDelete} disabled={!canDelete || saving || deleting}>
        {deleting ? 'Deleting…' : 'Delete'}
      </Button>
    </div>
  </div>

  <div class="space-y-5 px-5 py-5">
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

    <div class="space-y-2">
      <Label for="repo-clone-path">Clone path</Label>
      <Input
        id="repo-clone-path"
        value={draft.clonePath}
        placeholder="services/backend"
        oninput={(event) => updateTextField('clonePath', event)}
      />
      <p class="text-muted-foreground text-xs">
        Leave empty to let the runtime derive the workspace path automatically.
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
  </div>
</div>
