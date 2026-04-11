<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { CodeEditor, CodeViewer, DiffViewer } from '$lib/components/code'
  import { FileCode2 } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceFilePatch,
    ProjectConversationWorkspaceFilePreview,
    ProjectConversationWorkspaceRepoMetadata,
  } from '$lib/api/chat'
  import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state.svelte'
  import {
    type WorkspaceFileViewMode,
    workspaceFileReadOnlyMessage,
  } from './project-conversation-workspace-file-drafts'

  let {
    selectedRepo = null,
    selectedFilePath = '',
    preview = null,
    patch = null,
    editorState = null,
    draftDiff = '',
    fileLoading = false,
    fileError = '',
    runtimeActive = false,
    onViewModeChange,
    onDraftChange,
    onSave,
    onRevert,
    onKeepDraft,
    onReloadSavedVersion,
  }: {
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedFilePath?: string
    preview?: ProjectConversationWorkspaceFilePreview | null
    patch?: ProjectConversationWorkspaceFilePatch | null
    editorState?: WorkspaceFileEditorState | null
    draftDiff?: string
    fileLoading?: boolean
    fileError?: string
    runtimeActive?: boolean
    onViewModeChange?: (viewMode: WorkspaceFileViewMode) => void
    onDraftChange?: (value: string) => void
    onSave?: () => void
    onRevert?: () => void
    onKeepDraft?: () => void
    onReloadSavedVersion?: () => void
  } = $props()

  const fileName = $derived(selectedFilePath.split('/').pop() ?? '')
  const fileDirPath = $derived.by(() => {
    const parts = selectedFilePath.split('/')
    return parts.length > 1 ? parts.slice(0, -1).join('/') : ''
  })
  const viewMode = $derived(editorState?.viewMode ?? 'preview')
  const readOnlyMessage = $derived(
    preview?.writable === false ? workspaceFileReadOnlyMessage(preview.readOnlyReason) : '',
  )
  const fileState = $derived.by(() => {
    if (!editorState) {
      return { label: 'Saved', className: 'bg-emerald-500/10 text-emerald-700' }
    }
    if (editorState.savePhase === 'saving') {
      return { label: 'Saving...', className: 'bg-sky-500/10 text-sky-700' }
    }
    if (editorState.savePhase === 'conflict') {
      return { label: 'Conflict', className: 'bg-amber-500/15 text-amber-700' }
    }
    if (editorState.externalChange) {
      return { label: 'Changed in workspace', className: 'bg-amber-500/15 text-amber-700' }
    }
    if (editorState.dirty) {
      return { label: 'Unsaved', className: 'bg-orange-500/15 text-orange-700' }
    }
    return { label: 'Saved', className: 'bg-emerald-500/10 text-emerald-700' }
  })
</script>

<div class="flex h-full min-h-0 flex-col overflow-hidden">
  {#if !selectedRepo}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Select a repo to browse its files.
    </div>
  {:else if !selectedFilePath}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      <div class="space-y-2">
        <FileCode2 class="text-muted-foreground/30 mx-auto size-10" />
        <p>Select a file to view its contents</p>
      </div>
    </div>
  {:else if fileError}
    <div class="border-destructive/20 bg-destructive/5 m-4 rounded-lg border p-3">
      <p class="text-destructive text-sm">{fileError}</p>
    </div>
  {:else}
    <div class="border-border bg-muted/30 flex flex-wrap items-center gap-2 border-b px-3 py-2">
      <FileCode2 class="text-muted-foreground size-3.5 shrink-0" />
      <span class="min-w-0 truncate text-[13px] font-medium">{fileName}</span>
      {#if fileDirPath}
        <span class="text-muted-foreground/40 min-w-0 truncate text-[11px]">{fileDirPath}</span>
      {/if}
      <span class={cn('rounded px-1.5 py-0.5 text-[10px] font-semibold', fileState.className)}>
        {fileState.label}
      </span>
      {#if patch?.status && patch.status !== 'modified'}
        <span
          class={cn(
            'rounded px-1 text-[10px] font-bold uppercase',
            patch.status === 'added'
              ? 'bg-emerald-500/15 text-emerald-600'
              : patch.status === 'deleted'
                ? 'bg-rose-500/15 text-rose-600'
                : 'bg-sky-500/15 text-sky-600',
          )}
        >
          {patch.status}
        </span>
      {/if}
      <div class="ml-auto flex flex-wrap items-center gap-1">
        <Button
          size="sm"
          variant={viewMode === 'preview' ? 'secondary' : 'ghost'}
          onclick={() => onViewModeChange?.('preview')}
        >
          Preview
        </Button>
        <Button
          size="sm"
          variant={viewMode === 'edit' ? 'secondary' : 'ghost'}
          disabled={!preview || preview.previewKind !== 'text' || !preview.writable}
          onclick={() => onViewModeChange?.('edit')}
        >
          Edit
        </Button>
        <Button
          size="sm"
          variant={viewMode === 'diff' ? 'secondary' : 'ghost'}
          disabled={!editorState || draftDiff.length === 0}
          onclick={() => onViewModeChange?.('diff')}
        >
          Diff
        </Button>
        <Button
          size="sm"
          disabled={!editorState?.dirty || !preview?.writable || editorState.savePhase === 'saving'}
          onclick={() => onSave?.()}
        >
          Save
        </Button>
        <Button
          size="sm"
          variant="ghost"
          disabled={!editorState?.dirty && !editorState?.externalChange}
          onclick={() => onRevert?.()}
        >
          Revert
        </Button>
      </div>
      {#if preview}
        <span class="text-muted-foreground/50 text-[10px]">
          {preview.mediaType} · {preview.sizeBytes} B
        </span>
      {/if}
      {#if fileLoading}
        <span class="text-muted-foreground/50 text-[10px]">Loading…</span>
      {/if}
    </div>

    {#if runtimeActive}
      <div class="bg-muted/40 text-muted-foreground border-border border-b px-3 py-1.5 text-[11px]">
        Project AI can keep updating this workspace during active turns. Your local draft stays
        preserved.
      </div>
    {/if}

    {#if readOnlyMessage}
      <div class="border-border bg-muted/40 px-3 py-2 text-sm">{readOnlyMessage}</div>
    {/if}

    {#if editorState?.externalChange || editorState?.savePhase === 'conflict'}
      <div
        class="flex flex-wrap items-center gap-2 border-b border-amber-500/20 bg-amber-500/10 px-3 py-2 text-sm"
      >
        <span class="text-amber-900">
          {editorState.errorMessage ||
            'This file changed in the workspace while your draft was open.'}
        </span>
        <Button size="sm" variant="ghost" onclick={() => onViewModeChange?.('diff')}>
          Review changes
        </Button>
        <Button size="sm" variant="ghost" onclick={() => onReloadSavedVersion?.()}>
          Reload saved version
        </Button>
        <Button size="sm" variant="ghost" onclick={() => onKeepDraft?.()}>Keep my draft</Button>
      </div>
    {:else if editorState?.errorMessage}
      <div class="border-destructive/20 bg-destructive/5 border-b px-3 py-2 text-sm">
        <span class="text-destructive">{editorState.errorMessage}</span>
      </div>
    {/if}

    <div class="min-h-0 flex-1 overflow-hidden" data-testid="workspace-browser-detail-content">
      {#if preview?.previewKind === 'binary'}
        <div class="text-muted-foreground h-full overflow-auto px-4 py-8 text-center text-sm">
          <div class="mx-auto max-w-md">Binary file — not rendered inline.</div>
        </div>
      {:else if viewMode === 'edit' && editorState}
        <div
          class="h-full min-h-0 min-w-0 overflow-hidden"
          data-testid="workspace-browser-detail-scroll-frame"
        >
          <CodeEditor
            value={editorState.draftContent}
            filePath={selectedFilePath}
            readonly={!preview?.writable}
            class="h-full"
            onchange={(value) => onDraftChange?.(value)}
          />
        </div>
      {:else if viewMode === 'diff' && editorState}
        <div
          class="h-full min-h-0 min-w-0 overflow-hidden"
          data-testid="workspace-browser-detail-scroll-frame"
        >
          <DiffViewer
            diff={draftDiff}
            sourceContent={editorState.latestSavedContent}
            class="h-full"
          />
        </div>
      {:else if preview}
        <div
          class="h-full min-h-0 min-w-0 overflow-hidden"
          data-testid="workspace-browser-detail-scroll-frame"
        >
          <CodeViewer code={preview.content ?? ''} filePath={selectedFilePath} class="h-full" />
        </div>
      {:else if fileLoading}
        <div class="text-muted-foreground h-full overflow-auto px-4 py-8 text-center text-sm">
          Loading…
        </div>
      {:else}
        <div class="text-muted-foreground h-full overflow-auto px-4 py-8 text-center text-sm">
          Select a file to view its contents.
        </div>
      {/if}
    </div>
  {/if}
</div>
