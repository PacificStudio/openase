<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import {
    commitProjectConversationWorkspace,
    createProjectConversationWorkspaceBranch,
    discardProjectConversationWorkspaceFile,
    runProjectConversationWorkspaceGitRemoteOp,
    stageAllProjectConversationWorkspaceFiles,
    stageProjectConversationWorkspaceFile,
    syncProjectConversationWorkspace,
    unstageProjectConversationWorkspace,
  } from '$lib/api/chat'
  import type {
    ProjectConversationWorkspaceDiff,
    ProjectConversationWorkspaceGitRemoteOp,
  } from '$lib/api/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { AlertCircle } from '@lucide/svelte'
  import { PROJECT_AI_FOCUS_PRIORITY } from './project-ai-focus'
  import ProjectConversationWorkspaceBrowserPane from './project-conversation-workspace-browser-pane.svelte'
  import ProjectConversationWorkspaceBrowserToolbar from './project-conversation-workspace-browser-toolbar.svelte'
  import ProjectConversationWorkspaceSyncBanner from './project-conversation-workspace-sync-banner.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import { chatT } from './i18n'
  import { createTerminalManager } from './terminal-manager.svelte'
  import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    runtimeActive = false,
    syncGeneration = 0,
    pendingFilePath = '',
    onClose,
    onPendingFileConsumed,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    runtimeActive?: boolean
    syncGeneration?: number
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
    getWorkspaceDiff: () => liveWorkspaceDiff,
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
  let refreshGeneration = $state(0)
  let lastRefreshKey = $state('')
  let lastWorkspaceDiffLoading = $state(false)
  let lastConversationId = $state('')
  let lastSyncGeneration = $state(0)
  let syncInFlight = $state(false)
  let syncError = $state('')

  const selectedRepo = $derived(
    browser.metadata?.repos.find((repo) => repo.path === browser.selectedRepoPath) ??
      browser.metadata?.repos[0] ??
      null,
  )
  const selectedRepoDiff = $derived(
    liveWorkspaceDiff?.repos.find((repo) => repo.path === browser.selectedRepoPath) ?? null,
  )
  const syncPrompt = $derived(liveWorkspaceDiff?.syncPrompt ?? browser.metadata?.syncPrompt ?? null)

  function copyWorkspacePath() {
    const path = browser.metadata?.workspacePath
    if (!path) return
    navigator.clipboard.writeText(path)
    pathCopied = true
    setTimeout(() => (pathCopied = false), 1500)
  }

  async function handleSyncWorkspace() {
    if (!conversationId || syncInFlight) {
      return
    }
    syncInFlight = true
    syncError = ''
    try {
      await syncProjectConversationWorkspace(conversationId)
      workspaceBrowserPortal.markWorkspaceSynced()
      await Promise.resolve(workspaceBrowserPortal.onSyncWorkspace?.())
      await browser.refreshWorkspace(true)
    } catch (error) {
      syncError = error instanceof Error ? error.message : chatT('chat.failedToSyncWorkspace')
    } finally {
      syncInFlight = false
    }
  }

  async function handleGitRemoteOp(op: ProjectConversationWorkspaceGitRemoteOp) {
    if (!conversationId || !browser.selectedRepoPath) return
    await runProjectConversationWorkspaceGitRemoteOp(conversationId, {
      repoPath: browser.selectedRepoPath,
      op,
    })
    await browser.refreshRepoGitContext()
    await browser.refreshWorkspace(true)
  }

  async function refreshWorkspaceAfterGitMutation(
    repoPath: string,
    filePath = browser.selectedFilePath,
  ) {
    await browser.refreshRepoGitContext(repoPath)
    await browser.refreshWorkspace(true)
    await browser.refreshWorkspaceDiff()
    if (repoPath === browser.selectedRepoPath && filePath) {
      await browser.reloadFile(repoPath, filePath)
    }
  }

  async function handleStageFile(path: string) {
    if (!conversationId || !browser.selectedRepoPath || !path) return
    await stageProjectConversationWorkspaceFile(conversationId, {
      repoPath: browser.selectedRepoPath,
      path,
    })
    await refreshWorkspaceAfterGitMutation(browser.selectedRepoPath, path)
  }

  async function handleStageAll() {
    if (!conversationId || !browser.selectedRepoPath) return
    await stageAllProjectConversationWorkspaceFiles(conversationId, {
      repoPath: browser.selectedRepoPath,
    })
    await refreshWorkspaceAfterGitMutation(browser.selectedRepoPath)
  }

  async function handleUnstage(path = '') {
    if (!conversationId || !browser.selectedRepoPath) return
    await unstageProjectConversationWorkspace(conversationId, {
      repoPath: browser.selectedRepoPath,
      path,
    })
    await refreshWorkspaceAfterGitMutation(browser.selectedRepoPath, path)
  }

  async function handleCommitRepo(message: string) {
    if (!conversationId || !browser.selectedRepoPath) return
    await commitProjectConversationWorkspace(conversationId, {
      repoPath: browser.selectedRepoPath,
      message,
    })
    await refreshWorkspaceAfterGitMutation(browser.selectedRepoPath)
  }

  async function handleDiscardFile(path: string) {
    if (!conversationId || !browser.selectedRepoPath || !path) return
    const editorState = browser.getEditorState(browser.selectedRepoPath, path)
    const confirmMessage =
      editorState?.dirty === true
        ? `Discard all workspace changes for ${path}? Your unsaved editor draft will also be discarded.`
        : `Discard all workspace changes for ${path}?`
    if (!window.confirm(confirmMessage)) return
    if (editorState?.dirty === true) {
      browser.discardDraft(browser.selectedRepoPath, path)
    }
    await discardProjectConversationWorkspaceFile(conversationId, {
      repoPath: browser.selectedRepoPath,
      path,
    })
    await refreshWorkspaceAfterGitMutation(browser.selectedRepoPath, path)
  }

  function handleCreateBranch(commitId: string) {
    const name = window.prompt('New branch name:', '')
    if (!name?.trim() || !conversationId || !browser.selectedRepoPath) return
    void (async () => {
      await createProjectConversationWorkspaceBranch(conversationId, {
        repoPath: browser.selectedRepoPath,
        branchName: name.trim(),
        startPoint: commitId,
      })
      await browser.refreshRepoGitContext()
    })()
  }

  async function handleCreateBranchName(branchName: string) {
    if (!conversationId || !browser.selectedRepoPath || !branchName.trim()) return
    await createProjectConversationWorkspaceBranch(conversationId, {
      repoPath: browser.selectedRepoPath,
      branchName: branchName.trim(),
    })
    await browser.refreshRepoGitContext()
  }

  $effect(() => {
    if (pendingFilePath && browser.metadata?.available) {
      untrack(() => {
        browser.selectFile(pendingFilePath)
        onPendingFileConsumed?.()
      })
    }
  })

  $effect(() => {
    const pendingPatch = workspaceBrowserPortal.pendingPatch
    if (!pendingPatch || !browser.metadata?.available) {
      return
    }

    queueMicrotask(() => {
      const consumedPatch = workspaceBrowserPortal.consumePendingPatch()
      if (!consumedPatch) {
        return
      }
      void browser.reviewPatch(consumedPatch.diff, { autoApply: consumedPatch.autoApply })
    })
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
      lastSyncGeneration = syncGeneration
      return
    }
    if (syncGeneration !== lastSyncGeneration) {
      lastSyncGeneration = syncGeneration
      untrack(() => {
        void browser.refreshWorkspace(true)
      })
    }
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
      }
    }

    window.addEventListener('keydown', handleKeydown)
    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
  })

  $effect(() => {
    if (typeof window === 'undefined') {
      return
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!browser.hasDirtyTabs) {
        return
      }
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    const editorState = browser.selectedEditorState
    const focusContext = browser.getSelectedFocusContext()
    if (
      !projectId ||
      !conversationId ||
      !browser.selectedRepoPath ||
      !browser.selectedFilePath ||
      !editorState ||
      !focusContext
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
        selectedArea: focusContext.selectedArea,
        hasDirtyDraft: editorState.dirty,
        draftContent: editorState.dirty ? editorState.draftContent : undefined,
        encoding: editorState.encoding,
        lineEnding: editorState.lineEnding,
        selection: focusContext.selection,
        workingSet: focusContext.workingSet,
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
  <ProjectConversationWorkspaceBrowserToolbar
    workspacePath={browser.metadata?.workspacePath ?? ''}
    {pathCopied}
    showTerminalButton={Boolean(browser.metadata?.available)}
    terminalPanelOpen={terminalManager.panelOpen}
    {conversationId}
    metadataLoading={browser.metadataLoading}
    onCopyWorkspacePath={copyWorkspacePath}
    onToggleTerminal={() => terminalManager.togglePanel()}
    onRefreshWorkspace={() => void browser.refreshWorkspace(true)}
    {onClose}
  />

  {#if !conversationId}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      {chatT('chat.workspaceBrowserNoConversation')}
    </div>
  {:else if browser.metadataLoading && !browser.metadata}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      {chatT('chat.workspaceLoading')}
    </div>
  {:else if browser.metadataError}
    <div class="flex flex-1 items-center justify-center px-6">
      <div class="max-w-sm space-y-3 text-center">
        <div
          class="bg-destructive/10 text-destructive mx-auto flex size-10 items-center justify-center rounded-full"
        >
          <AlertCircle class="size-4" />
        </div>
        <p class="text-sm font-medium">{chatT('chat.workspaceBrowserUnavailable')}</p>
        <p class="text-muted-foreground text-sm">{browser.metadataError}</p>
      </div>
    </div>
  {:else if !browser.metadata?.available}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      {chatT('chat.workspaceProvisionNotice')}
    </div>
  {:else if syncPrompt && (browser.metadata?.repos.length ?? 0) === 0}
    <ProjectConversationWorkspaceSyncBanner
      prompt={syncPrompt}
      {syncError}
      {syncInFlight}
      centered
      onSync={handleSyncWorkspace}
    />
  {:else}
    <div class="flex min-h-0 flex-1 flex-col">
      {#if syncPrompt}
        <ProjectConversationWorkspaceSyncBanner
          prompt={syncPrompt}
          {syncError}
          {syncInFlight}
          onSync={handleSyncWorkspace}
        />
      {/if}
      <ProjectConversationWorkspaceBrowserPane
        {browser}
        {selectedRepo}
        {selectedRepoDiff}
        {runtimeActive}
        {terminalManager}
        currentRef={browser.repoRefs?.currentRef ?? selectedRepo?.currentRef ?? null}
        localBranches={browser.repoRefs?.localBranches ?? []}
        remoteBranches={browser.repoRefs?.remoteBranches ?? []}
        repoRefsLoading={browser.repoRefsLoading}
        repoRefsError={browser.repoRefsError}
        checkoutBlockers={browser.checkoutBlockers(browser.selectedRepoPath)}
        onCheckoutBranch={(request) =>
          browser.checkoutBranch({
            repoPath: browser.selectedRepoPath,
            ...request,
          })}
        onCreateBranchName={handleCreateBranchName}
        onGitRemoteOp={handleGitRemoteOp}
        onStageFile={handleStageFile}
        onStageAll={handleStageAll}
        onUnstage={handleUnstage}
        onCommitRepo={handleCommitRepo}
        onDiscardFile={handleDiscardFile}
        onCreateBranch={handleCreateBranch}
      />
    </div>
  {/if}
</div>
