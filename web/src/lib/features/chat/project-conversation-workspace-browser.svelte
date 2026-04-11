<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    AlertCircle,
    Check,
    Copy,
    FolderTree,
    RefreshCcw,
    SquareTerminal,
    X,
  } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import { PROJECT_AI_FOCUS_PRIORITY } from './project-ai-focus'
  import ProjectConversationWorkspaceBrowserPane from './project-conversation-workspace-browser-pane.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import { createTerminalManager } from './terminal-manager.svelte'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    runtimeActive = false,
    pendingFilePath = '',
    onClose,
    onPendingFileConsumed,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    runtimeActive?: boolean
    /** File path to navigate to (consumed once on change). */
    pendingFilePath?: string
    onClose?: () => void
    onPendingFileConsumed?: () => void
  } = $props()

  const projectAIFocusOwner = 'project-conversation-workspace-browser'
  let refreshedWorkspaceDiff = $state<ProjectConversationWorkspaceDiff | null>(null)
  const liveWorkspaceDiff = $derived(refreshedWorkspaceDiff ?? workspaceDiff ?? null)

  const browser = createProjectConversationWorkspaceBrowserState({
    getConversationId: () => conversationId,
    onWorkspaceDiffUpdated: (nextWorkspaceDiff) => {
      refreshedWorkspaceDiff = nextWorkspaceDiff
    },
  })

  const terminalManager = createTerminalManager({
    getConversationId: () => conversationId,
    getWorkspacePath: () => browser.metadata?.workspacePath ?? '',
  })

  onDestroy(() => {
    terminalManager.disposeAll()
  })

  let pathCopied = $state(false)

  function copyWorkspacePath() {
    const path = browser.metadata?.workspacePath
    if (!path) return
    navigator.clipboard.writeText(path)
    pathCopied = true
    setTimeout(() => (pathCopied = false), 1500)
  }

  let refreshGeneration = $state(0)
  let lastRefreshKey = $state('')
  let lastWorkspaceDiffLoading = $state(false)
  let lastConversationId = $state('')

  const selectedRepo = $derived(
    browser.metadata?.repos.find((repo) => repo.path === browser.selectedRepoPath) ??
      browser.metadata?.repos[0] ??
      null,
  )
  const selectedRepoDiff = $derived(
    liveWorkspaceDiff?.repos.find((repo) => repo.path === browser.selectedRepoPath) ?? null,
  )

  // Navigate to a pending file when set
  $effect(() => {
    if (pendingFilePath && browser.metadata?.available) {
      untrack(() => {
        browser.selectFile(pendingFilePath)
        onPendingFileConsumed?.()
      })
    }
  })

  $effect(() => {
    if (lastConversationId && lastConversationId !== conversationId) {
      terminalManager.disposeAll()
    }
    lastConversationId = conversationId
  })

  $effect(() => {
    const nextLoading = workspaceDiffLoading
    if (lastWorkspaceDiffLoading && !nextLoading && conversationId) {
      refreshGeneration += 1
    }
    lastWorkspaceDiffLoading = nextLoading
  })

  $effect(() => {
    if (!conversationId) {
      lastRefreshKey = ''
      browser.reset()
      terminalManager.disposeAll()
      return
    }

    const refreshKey = `${conversationId}:${refreshGeneration}`
    if (lastRefreshKey === refreshKey) {
      return
    }
    lastRefreshKey = refreshKey

    untrack(() => {
      void browser.refreshWorkspace(true)
    })
  })

  $effect(() => {
    if (typeof window === 'undefined') {
      return
    }

    const handleKeydown = (event: KeyboardEvent) => {
      const editorState = browser.selectedEditorState
      if (!editorState) {
        return
      }

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        if (!editorState.dirty || !browser.preview?.writable) {
          return
        }
        event.preventDefault()
        void browser.saveSelectedFile()
        return
      }

      if (event.key === 'Escape' && editorState.viewMode === 'edit') {
        event.preventDefault()
        browser.setSelectedViewMode('preview')
      }
    }

    window.addEventListener('keydown', handleKeydown)
    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    const editorState = browser.selectedEditorState
    if (
      !projectId ||
      !conversationId ||
      !browser.selectedRepoPath ||
      !browser.selectedFilePath ||
      !editorState
    ) {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
      return
    }

    appStore.setProjectAssistantFocus(
      projectAIFocusOwner,
      {
        kind: 'workspace_file',
        projectId,
        conversationId,
        repoPath: browser.selectedRepoPath,
        filePath: browser.selectedFilePath,
        selectedArea: editorState.viewMode,
        hasDirtyDraft: editorState.dirty,
        draftContent: editorState.dirty ? editorState.draftContent : undefined,
        encoding: editorState.encoding,
        lineEnding: editorState.lineEnding,
      },
      PROJECT_AI_FOCUS_PRIORITY.workspace,
    )

    return () => {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
    }
  })
</script>

<div
  class="bg-background flex h-full min-h-0 w-full flex-col"
  data-testid="project-conversation-workspace-browser"
>
  <!-- Compact toolbar -->
  <div class="border-border flex h-9 items-center gap-1.5 border-b px-3">
    <FolderTree class="text-muted-foreground size-3 shrink-0" />
    <span class="text-[12px] font-semibold">Workspace</span>
    {#if browser.metadata?.workspacePath}
      <button
        type="button"
        class="text-muted-foreground/50 hover:text-muted-foreground group flex min-w-0 items-center gap-1 truncate text-[11px] transition-colors"
        title="Click to copy path"
        onclick={copyWorkspacePath}
      >
        <span class="min-w-0 truncate">{browser.metadata.workspacePath}</span>
        {#if pathCopied}
          <Check class="size-2.5 shrink-0 text-emerald-500" />
        {:else}
          <Copy class="size-2.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
        {/if}
      </button>
    {/if}
    <div class="flex-1"></div>
    {#if browser.metadata?.available}
      <Button
        variant={terminalManager.panelOpen ? 'secondary' : 'ghost'}
        size="icon-xs"
        class={cn('text-muted-foreground size-6', terminalManager.panelOpen && 'text-foreground')}
        aria-label="Toggle terminal"
        onclick={() => terminalManager.togglePanel()}
        disabled={!conversationId}
      >
        <SquareTerminal class="size-3" />
      </Button>
    {/if}
    <Button
      variant="ghost"
      size="icon-xs"
      class="text-muted-foreground size-6"
      aria-label="Refresh workspace browser"
      onclick={() => void browser.refreshWorkspace(true)}
      disabled={!conversationId || browser.metadataLoading}
    >
      <RefreshCcw class={cn('size-3', browser.metadataLoading && 'animate-spin')} />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="icon-xs"
        class="text-muted-foreground size-6"
        aria-label="Close workspace browser"
        onclick={onClose}
      >
        <X class="size-3" />
      </Button>
    {/if}
  </div>

  {#if !conversationId}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Start or reopen a Project AI conversation to inspect its workspace.
    </div>
  {:else if browser.metadataLoading && !browser.metadata}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Loading workspace...
    </div>
  {:else if browser.metadataError}
    <div class="flex flex-1 items-center justify-center px-6">
      <div class="max-w-sm space-y-3 text-center">
        <div
          class="bg-destructive/10 text-destructive mx-auto flex size-10 items-center justify-center rounded-full"
        >
          <AlertCircle class="size-4" />
        </div>
        <p class="text-sm font-medium">Workspace browser unavailable</p>
        <p class="text-muted-foreground text-sm">{browser.metadataError}</p>
      </div>
    </div>
  {:else if !browser.metadata?.available}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      The workspace will appear after Project AI provisions the conversation workdir.
    </div>
  {:else}
    <ProjectConversationWorkspaceBrowserPane
      {browser}
      {selectedRepo}
      {selectedRepoDiff}
      {runtimeActive}
      {terminalManager}
    />
  {/if}
</div>
