<script lang="ts">
  /* eslint-disable max-lines */
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { cn } from '$lib/utils'
  import { CodeEditor } from '$lib/components/code'
  import { readEditorWrapMode, storeEditorWrapMode } from '$lib/components/code/wrap-mode'
  import { FileCode2, WrapText, X } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceRepoMetadata } from '$lib/api/chat'
  import {
    workspaceTabKey,
    type ProjectConversationWorkspaceBrowserState,
  } from './project-conversation-workspace-browser-state.svelte'
  import {
    workspaceFileStateClass,
    workspaceFileStateLabel,
  } from './project-conversation-workspace-browser-detail-state'
  import { workspaceFileReadOnlyMessage } from './project-conversation-workspace-file-drafts'

  let {
    browser,
    selectedRepo = null,
    runtimeActive = false,
  }: {
    browser: ProjectConversationWorkspaceBrowserState
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    runtimeActive?: boolean
  } = $props()

  let pendingClose = $state<{ repoPath: string; filePath: string } | null>(null)
  let saving = $state(false)
  let wrapMode = $state(readEditorWrapMode())
  const dialogOpen = $derived(pendingClose !== null)

  function isTabDirty(repoPath: string, filePath: string): boolean {
    return browser.getEditorState(repoPath, filePath)?.dirty === true
  }

  function requestCloseTab(event: MouseEvent, repoPath: string, filePath: string) {
    event.stopPropagation()
    if (isTabDirty(repoPath, filePath)) {
      pendingClose = { repoPath, filePath }
      return
    }
    browser.discardDraft(repoPath, filePath)
    browser.closeTab(repoPath, filePath)
  }

  function dismissDialog() {
    pendingClose = null
  }

  async function confirmSaveAndClose() {
    if (!pendingClose) return
    const target = pendingClose
    saving = true
    try {
      browser.activateTab(target.repoPath, target.filePath)
      const ok = await browser.saveFile(target.repoPath, target.filePath)
      if (ok) {
        browser.closeTab(target.repoPath, target.filePath)
        pendingClose = null
      }
    } finally {
      saving = false
    }
  }

  function confirmDiscardAndClose() {
    if (!pendingClose) return
    const target = pendingClose
    browser.discardDraft(target.repoPath, target.filePath)
    browser.closeTab(target.repoPath, target.filePath)
    pendingClose = null
  }

  const pendingCloseFilename = () => pendingClose?.filePath.split('/').pop() ?? ''

  function toggleWrapMode() {
    wrapMode = wrapMode === 'wrap' ? 'nowrap' : 'wrap'
    storeEditorWrapMode(wrapMode)
  }

  const activeFilePath = $derived(
    browser.openTabs.find((tab) => workspaceTabKey(tab) === browser.activeTabKey)?.filePath ?? '',
  )
  const activeFileName = $derived(activeFilePath.split('/').pop() ?? '')
  const activeFileDirPath = $derived.by(() => {
    const parts = activeFilePath.split('/')
    return parts.length > 1 ? parts.slice(0, -1).join('/') : ''
  })
  const activePreview = $derived(browser.preview)
  const activePatch = $derived(browser.patch)
  const activeFileLoading = $derived(browser.fileLoading)
  const activeFileError = $derived(browser.fileError)
  const activeEditorState = $derived(browser.selectedEditorState)
  const activeDiffMarkers = $derived(browser.selectedDraftLineDiff)
  const readOnlyMessage = $derived(
    activePreview?.writable === false
      ? workspaceFileReadOnlyMessage(activePreview.readOnlyReason)
      : '',
  )
  const showWrapToggle = $derived(activePreview?.previewKind === 'text' && !!activeEditorState)
</script>

<div class="flex h-full min-h-0 flex-col overflow-hidden">
  {#if !selectedRepo}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Select a repo to browse its files.
    </div>
  {:else if browser.openTabs.length === 0}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      <div class="space-y-2">
        <FileCode2 class="text-muted-foreground/30 mx-auto size-10" />
        <p>Select a file to view its contents</p>
      </div>
    </div>
  {:else}
    <div
      class="border-border bg-muted/20 flex min-h-9 shrink-0 items-stretch overflow-x-auto border-b"
      data-testid="workspace-browser-detail-tab-bar"
      role="tablist"
    >
      {#each browser.openTabs as tab (workspaceTabKey(tab))}
        {@const tabKey = workspaceTabKey(tab)}
        {@const isActive = tabKey === browser.activeTabKey}
        {@const dirty = isTabDirty(tab.repoPath, tab.filePath)}
        {@const tabName = tab.filePath.split('/').pop() ?? tab.filePath}
        <button
          type="button"
          role="tab"
          aria-selected={isActive}
          class={cn(
            'border-border flex max-w-[220px] min-w-0 shrink-0 items-center gap-1.5 border-r px-2.5 py-1.5 text-[12px] transition-colors',
            isActive
              ? 'bg-background text-foreground'
              : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
          )}
          onclick={() => browser.activateTab(tab.repoPath, tab.filePath)}
          data-testid={`workspace-browser-detail-tab-${tabName}`}
        >
          <FileCode2 class="size-3 shrink-0 opacity-60" />
          <span class="min-w-0 truncate">{tabName}</span>
          <span
            role="button"
            tabindex="0"
            class="hover:bg-muted/80 ml-0.5 inline-flex size-4 shrink-0 items-center justify-center rounded transition-colors"
            aria-label={`Close ${tabName}`}
            onclick={(event) => requestCloseTab(event, tab.repoPath, tab.filePath)}
            onkeydown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                event.stopPropagation()
                requestCloseTab(event as unknown as MouseEvent, tab.repoPath, tab.filePath)
              }
            }}
          >
            {#if dirty}
              <span
                class="size-1.5 rounded-full bg-orange-500"
                aria-label="Unsaved changes"
                data-testid="workspace-browser-detail-tab-dirty-dot"
              ></span>
            {:else}
              <X class="size-3" />
            {/if}
          </span>
        </button>
      {/each}
    </div>

    {#if activeFileError}
      <div class="border-destructive/20 bg-destructive/5 m-4 rounded-lg border p-3">
        <p class="text-destructive text-sm">{activeFileError}</p>
      </div>
    {:else if browser.activeTabKey}
      <div class="border-border bg-muted/30 flex flex-wrap items-center gap-2 border-b px-3 py-1.5">
        <FileCode2 class="text-muted-foreground size-3 shrink-0" />
        <span class="min-w-0 truncate text-[12px] font-medium">{activeFileName}</span>
        {#if activeFileDirPath}
          <span class="text-muted-foreground/40 min-w-0 truncate text-[11px]">
            {activeFileDirPath}
          </span>
        {/if}
        {#if activePatch?.status && activePatch.status !== 'modified'}
          <span
            class={cn(
              'rounded px-1 text-[10px] font-bold uppercase',
              activePatch.status === 'added'
                ? 'bg-emerald-500/15 text-emerald-600'
                : activePatch.status === 'deleted'
                  ? 'bg-rose-500/15 text-rose-600'
                  : 'bg-sky-500/15 text-sky-600',
            )}
          >
            {activePatch.status}
          </span>
        {/if}
        {#if activeEditorState}
          <span
            class={cn(
              'rounded px-2 py-0.5 text-[10px] font-medium',
              workspaceFileStateClass(activeEditorState),
            )}
          >
            {workspaceFileStateLabel(activeEditorState)}
          </span>
        {/if}
        <div class="ml-auto flex items-center gap-2">
          {#if showWrapToggle}
            <Button
              variant={wrapMode === 'wrap' ? 'secondary' : 'ghost'}
              size="icon-xs"
              aria-label={wrapMode === 'wrap' ? 'Disable line wrap' : 'Enable line wrap'}
              aria-pressed={wrapMode === 'wrap'}
              title={wrapMode === 'wrap' ? 'Disable line wrap' : 'Enable line wrap'}
              data-testid="workspace-browser-wrap-toggle"
              onclick={toggleWrapMode}
            >
              <WrapText />
            </Button>
          {/if}
          {#if activeEditorState}
            <Button
              size="sm"
              variant="ghost"
              disabled={!activeEditorState.dirty || activeEditorState.savePhase === 'saving'}
              onclick={() => browser.revertSelectedDraft()}
            >
              Revert
            </Button>
            <Button
              size="sm"
              disabled={!activeEditorState.dirty ||
                activeEditorState.savePhase === 'saving' ||
                activePreview?.writable !== true}
              onclick={() => void browser.saveSelectedFile()}
            >
              {activeEditorState.savePhase === 'saving' ? 'Saving...' : 'Save'}
            </Button>
          {/if}
          {#if activePreview}
            <span class="text-muted-foreground/50 text-[10px]">
              {activePreview.mediaType} · {activePreview.sizeBytes} B
            </span>
          {/if}
          {#if activeFileLoading}
            <span class="text-muted-foreground/50 text-[10px]">Loading…</span>
          {/if}
        </div>
      </div>

      {#if runtimeActive}
        <div
          class="bg-muted/40 text-muted-foreground border-border border-b px-3 py-1.5 text-[11px]"
        >
          Project AI can keep updating this workspace during active turns. Your local draft stays
          preserved.
        </div>
      {/if}

      {#if readOnlyMessage}
        <div class="border-border bg-muted/40 px-3 py-2 text-sm">{readOnlyMessage}</div>
      {/if}

      {#if activeEditorState?.externalChange || activeEditorState?.savePhase === 'conflict'}
        <div
          class="flex flex-wrap items-center gap-2 border-b border-amber-500/20 bg-amber-500/10 px-3 py-2 text-sm"
        >
          <span class="text-amber-900">
            {activeEditorState.errorMessage ||
              'This file changed in the workspace while your draft was open.'}
          </span>
          <Button size="sm" variant="ghost" onclick={() => browser.reloadSelectedSavedVersion()}>
            Reload saved version
          </Button>
          <Button size="sm" variant="ghost" onclick={() => browser.keepSelectedDraft()}>
            Keep my draft
          </Button>
        </div>
      {:else if activeEditorState?.errorMessage}
        <div class="border-destructive/20 bg-destructive/5 border-b px-3 py-2 text-sm">
          <span class="text-destructive">{activeEditorState.errorMessage}</span>
        </div>
      {/if}

      <div class="min-h-0 flex-1 overflow-hidden" data-testid="workspace-browser-detail-content">
        {#if activePreview?.previewKind === 'binary'}
          <div class="text-muted-foreground h-full overflow-auto px-4 py-8 text-center text-sm">
            <div class="mx-auto max-w-md">Binary file — not rendered inline.</div>
          </div>
        {:else if activeEditorState && activePreview}
          <div
            class="h-full min-h-0 min-w-0 overflow-hidden"
            data-testid="workspace-browser-detail-scroll-frame"
          >
            <CodeEditor
              value={activeEditorState.draftContent}
              filePath={activeFilePath}
              readonly={!activePreview.writable}
              {wrapMode}
              diffMarkers={activeDiffMarkers}
              class="h-full"
              onchange={(value) => browser.updateSelectedDraft(value)}
            />
          </div>
        {:else if activeFileLoading}
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
  {/if}
</div>

<Dialog.Root
  open={dialogOpen}
  onOpenChange={(next) => {
    if (!next) dismissDialog()
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Save changes?</Dialog.Title>
      <Dialog.Description>
        {pendingCloseFilename()} has unsaved changes. Save them before closing the tab?
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Button variant="ghost" onclick={dismissDialog} disabled={saving}>Cancel</Button>
      <Button variant="ghost" onclick={confirmDiscardAndClose} disabled={saving}>
        Don&apos;t save
      </Button>
      <Button onclick={confirmSaveAndClose} disabled={saving}>
        {saving ? 'Saving…' : 'Save'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
