<script lang="ts">
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import RepositoryEditor from './repository-editor.svelte'
  import type { RepositoryDraft, RepositoryEditorMode } from '../repositories-model'

  let {
    open = $bindable(false),
    mode,
    selectedRepo,
    draft,
    reposCount = 0,
    saving = false,
    onDraftChange,
    onSave,
  }: {
    open?: boolean
    mode: RepositoryEditorMode
    selectedRepo: ProjectRepoRecord | null
    draft: RepositoryDraft
    reposCount?: number
    saving?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
    onSave?: () => void
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-xl"
    data-testid="repository-editor-sheet"
  >
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <div class="flex items-start justify-between gap-4 pr-10">
        <div class="min-w-0 space-y-2">
          <SheetTitle>
            {mode === 'create' ? 'Add repository' : (selectedRepo?.name ?? 'Edit repository')}
          </SheetTitle>
          <SheetDescription>
            Configure repository metadata that ticket repo scopes, workflows, and workspace
            preparation consume.
          </SheetDescription>
        </div>

        <Button size="sm" onclick={onSave} disabled={saving} data-testid="repository-save-button">
          {saving ? 'Saving…' : mode === 'create' ? 'Create repository' : 'Save changes'}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 overflow-y-auto px-6 py-6">
      <RepositoryEditor {mode} {selectedRepo} {draft} {reposCount} {saving} {onDraftChange} />
    </div>
  </SheetContent>
</Sheet>
