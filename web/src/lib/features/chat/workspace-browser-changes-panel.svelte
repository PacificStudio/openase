<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ProjectConversationWorkspaceDiffRepo } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import { chatT } from './i18n'
  import { ChevronRight } from '@lucide/svelte'
  import WorkspaceBrowserChangeSection from './workspace-browser-change-section.svelte'

  let {
    commitHash = '',
    selectedRepoDiff = null,
    selectedFilePath = '',
    onStageFile,
    onStageAll,
    onUnstage,
    onCommitRepo,
    onDiscardFile,
    onSelectFile,
  }: {
    commitHash?: string
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    selectedFilePath?: string
    onStageFile?: (path: string) => Promise<void>
    onStageAll?: () => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onCommitRepo?: (message: string) => Promise<void>
    onDiscardFile?: (path: string) => Promise<void>
    onSelectFile?: (path: string) => void
  } = $props()

  let actionPendingPath = $state('')
  let actionError = $state('')
  let commitMessage = $state('')
  let commitPending = $state(false)
  let changesExpanded = $state(false)

  const diffFiles = $derived(selectedRepoDiff?.files ?? [])
  const hasDiff = $derived(selectedRepoDiff != null && selectedRepoDiff.filesChanged > 0)
  const stagedFiles = $derived(diffFiles.filter((file) => file.staged))
  const unstagedFiles = $derived(diffFiles.filter((file) => file.unstaged))
  const stagedFileCount = $derived(stagedFiles.length)
  const unstagedFileCount = $derived(unstagedFiles.length)
  const canCommit = $derived(
    stagedFileCount > 0 && commitMessage.trim().length > 0 && !commitPending,
  )

  function formatError(error: unknown, fallback: string) {
    return error instanceof ApiError
      ? error.detail
      : error instanceof Error
        ? error.message
        : fallback
  }

  async function handleStage(path: string) {
    if (!onStageFile) return
    actionPendingPath = path
    actionError = ''
    try {
      await onStageFile(path)
    } catch (error) {
      actionError = formatError(error, 'Failed to stage the file.')
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleStageAll() {
    if (!onStageAll) return
    actionPendingPath = '*'
    actionError = ''
    try {
      await onStageAll()
    } catch (error) {
      actionError = formatError(error, 'Failed to stage all files.')
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleUnstage(path = '') {
    if (!onUnstage) return
    actionPendingPath = path || '*'
    actionError = ''
    try {
      await onUnstage(path || undefined)
    } catch (error) {
      actionError = formatError(error, 'Failed to unstage changes.')
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleDiscard(path: string) {
    if (!onDiscardFile) return
    actionPendingPath = path
    actionError = ''
    try {
      await onDiscardFile(path)
    } catch (error) {
      actionError = formatError(error, 'Failed to discard the file changes.')
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleCommit() {
    if (!onCommitRepo || !canCommit) return
    commitPending = true
    actionError = ''
    try {
      await onCommitRepo(commitMessage.trim())
      commitMessage = ''
    } catch (error) {
      actionError = formatError(error, 'Failed to create the commit.')
    } finally {
      commitPending = false
    }
  }
</script>

<div class="border-border shrink-0 border-t">
  <button
    type="button"
    class="bg-muted/30 hover:bg-muted/60 flex w-full min-w-0 items-center gap-1.5 px-2 py-1.5 text-left text-[11px] transition-colors"
    onclick={() => (changesExpanded = !changesExpanded)}
  >
    {#if commitHash}
      <span class="text-muted-foreground/50 shrink-0 font-mono text-[10px]">{commitHash}</span>
    {/if}
    {#if hasDiff}
      <span class="text-muted-foreground/50 shrink-0">·</span>
      <span class="shrink-0 text-[10px] font-medium">
        {selectedRepoDiff?.filesChanged} file{selectedRepoDiff?.filesChanged === 1 ? '' : 's'}
      </span>
      {#if stagedFileCount > 0}
        <span class="shrink-0 text-[9px] text-emerald-600">{stagedFileCount}✓</span>
      {/if}
      <span class="shrink-0 text-[10px] text-emerald-600">+{selectedRepoDiff?.added ?? 0}</span>
      <span class="shrink-0 text-[10px] text-red-500">-{selectedRepoDiff?.removed ?? 0}</span>
      <ChevronRight
        class={cn(
          'text-muted-foreground ml-auto size-2.5 shrink-0 transition-transform duration-100',
          changesExpanded && 'rotate-90',
        )}
      />
    {:else}
      <span class="text-muted-foreground/40 text-[10px]"
        >{chatT('chat.workspace.changes.clean')}</span
      >
    {/if}
  </button>

  {#if changesExpanded && hasDiff}
    <div class="border-border max-h-64 overflow-y-auto border-t">
      <div class="bg-muted/20 flex items-end gap-1 px-2 py-1.5">
        <textarea
          class="border-input bg-background placeholder:text-muted-foreground/50 focus-visible:ring-ring min-w-0 flex-1 resize-none rounded border px-2 py-1 text-[11px] leading-snug outline-none focus-visible:ring-1"
          placeholder={chatT('chat.workspace.changes.commitPlaceholder')}
          rows="2"
          bind:value={commitMessage}
          data-testid="workspace-branch-commit-message"
          onkeydown={(event) => {
            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
              event.preventDefault()
              void handleCommit()
            }
          }}
        ></textarea>
        <button
          type="button"
          class={cn(
            'shrink-0 rounded px-2 py-1 text-[10px] font-semibold transition-colors',
            canCommit
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-muted text-muted-foreground cursor-not-allowed',
          )}
          data-testid="workspace-branch-commit-button"
          disabled={!canCommit}
          onclick={() => void handleCommit()}
        >
          {commitPending ? 'Committing...' : 'Commit'}
        </button>
      </div>

      {#if actionError}
        <div class="border-destructive/20 bg-destructive/5 border-t px-3 py-1 text-[11px]">
          <span class="text-destructive">{actionError}</span>
        </div>
      {/if}

      {#if stagedFileCount > 0}
        <WorkspaceBrowserChangeSection
          title={chatT('chat.workspace.changes.stagedTitle')}
          count={stagedFileCount}
          files={stagedFiles}
          {selectedFilePath}
          mode="staged"
          {actionPendingPath}
          {commitPending}
          headerActionTitle="Unstage all"
          headerActionTestId="workspace-branch-unstage-all"
          onHeaderAction={() => handleUnstage()}
          {onSelectFile}
          onUnstage={handleUnstage}
        />
      {/if}

      {#if unstagedFileCount > 0}
        <WorkspaceBrowserChangeSection
          title={chatT('chat.explorer.changesTitle')}
          count={unstagedFileCount}
          files={unstagedFiles}
          {selectedFilePath}
          mode="unstaged"
          {actionPendingPath}
          {commitPending}
          headerActionTitle="Stage all"
          headerActionTestId="workspace-branch-stage-all"
          onHeaderAction={handleStageAll}
          {onSelectFile}
          onStage={handleStage}
          onDiscard={handleDiscard}
        />
      {/if}
    </div>
  {/if}
</div>
