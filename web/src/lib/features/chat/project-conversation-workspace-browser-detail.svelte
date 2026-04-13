<script lang="ts">
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { CodeEditor } from '$lib/components/code'
  import { readEditorWrapMode, storeEditorWrapMode } from '$lib/components/code/wrap-mode'
  import { appStore } from '$lib/stores/app.svelte'
  import StructuredDiffPreview from './structured-diff-preview.svelte'
  import ProjectConversationWorkspaceBrowserTabStrip from './project-conversation-workspace-browser-tab-strip.svelte'
  import { FileCode2 } from '@lucide/svelte'
  import ProjectConversationWorkspaceBrowserDetailStatusBar from './project-conversation-workspace-browser-detail-status-bar.svelte'
  import type {
    ProjectConversationWorkspaceDiffFile,
    ProjectConversationWorkspaceRepoMetadata,
  } from '$lib/api/chat'
  import {
    workspaceTabKey,
    type ProjectConversationWorkspaceBrowserState,
  } from './project-conversation-workspace-browser-state.svelte'
  import { workspaceFileReadOnlyMessage } from './project-conversation-workspace-file-drafts'
  import {
    computePatchLineDiff,
    isWorkspaceFileLineDiffEmpty,
  } from './project-conversation-workspace-line-diff'

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
  const activePreview = $derived(browser.preview)
  const activePatch = $derived(browser.patch)
  const activeFileLoading = $derived(browser.fileLoading)
  const activeFileError = $derived(browser.fileError)
  const activeEditorState = $derived(browser.selectedEditorState)
  const activeDiffMarkers = $derived.by(() => {
    const draftDiff = browser.selectedDraftLineDiff
    if (!isWorkspaceFileLineDiffEmpty(draftDiff)) {
      return draftDiff
    }
    if (!activePatch || !activePreview) {
      return draftDiff
    }
    return computePatchLineDiff({
      status: activePatch.status,
      diffKind: activePatch.diffKind,
      diff: activePatch.diff,
      content: activePreview.content,
    })
  })
  const readOnlyMessage = $derived(
    activePreview?.writable === false
      ? workspaceFileReadOnlyMessage(activePreview.readOnlyReason)
      : '',
  )
  const showWrapToggle = $derived(activePreview?.previewKind === 'text' && !!activeEditorState)
  const selectedChangedFiles = $derived(browser.selectedChangedFiles)
  const activeChangedFile = $derived<ProjectConversationWorkspaceDiffFile | null>(
    selectedChangedFiles.find((file) => file.path === activeFilePath) ?? null,
  )
  const pendingPatch = $derived(activeEditorState?.pendingPatch ?? null)
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
    <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
      <ProjectConversationWorkspaceBrowserTabStrip
        openTabs={browser.openTabs}
        activeTabKey={browser.activeTabKey}
        {isTabDirty}
        onActivateTab={(repoPath, filePath) => browser.activateTab(repoPath, filePath)}
        onRequestCloseTab={requestCloseTab}
      />

      {#if activeFileError}
        <div class="border-destructive/20 bg-destructive/5 m-4 rounded-lg border p-3">
          <p class="text-destructive text-sm">{activeFileError}</p>
        </div>
      {:else if browser.activeTabKey}
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

        {#if pendingPatch}
          <div class="border-border bg-muted/20 space-y-3 border-b px-3 py-3">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-medium">Project AI patch proposal</span>
              <div class="ml-auto flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  onclick={() => browser.discardSelectedPendingPatch()}
                >
                  Discard patch
                </Button>
                <Button size="sm" onclick={() => browser.applySelectedPendingPatch()}>
                  Apply patch to editor
                </Button>
              </div>
            </div>
            <StructuredDiffPreview preview={pendingPatch.preview} />
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
                onselectionchange={(selection) => browser.updateSelectedSelection(selection)}
                onFormatDocument={activePreview.writable
                  ? () => browser.formatSelectedDocument()
                  : undefined}
                onFormatSelection={activePreview.writable
                  ? () => browser.formatSelectedSelection()
                  : undefined}
                onSave={activePreview.writable ? () => void browser.saveSelectedFile() : undefined}
                onRevert={activeEditorState.dirty && activeEditorState.savePhase !== 'saving'
                  ? () => browser.revertSelectedDraft()
                  : undefined}
                onExplainSelection={() =>
                  appStore.requestProjectAssistant('Explain the selected code.')}
                onRewriteSelection={() =>
                  appStore.requestProjectAssistant('Rewrite the selected code.')}
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

        <ProjectConversationWorkspaceBrowserDetailStatusBar
          {activeEditorState}
          {activePatch}
          {activeFileLoading}
          {activePreview}
          {activeChangedFile}
          selectedChangedFilesCount={selectedChangedFiles.length}
          {showWrapToggle}
          {wrapMode}
          autosaveEnabled={browser.autosaveEnabled}
          onToggleWrapMode={toggleWrapMode}
          onSelectPreviousChangedFile={() => browser.selectPreviousChangedFile()}
          onSelectNextChangedFile={() => browser.selectNextChangedFile()}
          onToggleAutosave={() => browser.setAutosaveEnabled(!browser.autosaveEnabled)}
        />
      {/if}
    </div>
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
