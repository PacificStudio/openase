<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { syncProjectConversationWorkspace } from '$lib/api/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { AlertCircle } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import { PROJECT_AI_FOCUS_PRIORITY } from './project-ai-focus'
  import ProjectConversationWorkspaceBrowserPane from './project-conversation-workspace-browser-pane.svelte'
  import ProjectConversationWorkspaceBrowserToolbar from './project-conversation-workspace-browser-toolbar.svelte'
  import ProjectConversationWorkspaceSyncBanner from './project-conversation-workspace-sync-banner.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
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
      syncError =
        error instanceof Error ? error.message : 'Failed to sync the Project AI workspace.'
    } finally {
      syncInFlight = false
    }
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
        selectedArea: 'edit',
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
      />
    </div>
  {/if}
</div>
