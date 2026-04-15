<script lang="ts">
  import { AlertCircle } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceGitRemoteOp,
    ProjectConversationWorkspaceRepoMetadata,
    ProjectConversationWorkspaceSyncPrompt,
  } from '$lib/api/chat'
  import type { ProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import ProjectConversationWorkspaceBrowserPane from './project-conversation-workspace-browser-pane.svelte'
  import ProjectConversationWorkspaceBrowserToolbar from './project-conversation-workspace-browser-toolbar.svelte'
  import ProjectConversationWorkspaceSyncBanner from './project-conversation-workspace-sync-banner.svelte'
  import { chatT } from './i18n'
  import { createTerminalManager } from './terminal-manager.svelte'

  let {
    conversationId = '',
    browser,
    syncPrompt = null,
    syncError = '',
    syncInFlight = false,
    selectedRepo = null,
    selectedRepoDiff = null,
    runtimeActive = false,
    terminalManager,
    onRefreshWorkspace,
    onSyncWorkspace,
    onClose,
    onCheckoutBranch,
    onCreateBranchName,
    onGitRemoteOp,
    onStageFile,
    onStageAll,
    onUnstage,
    onCommitRepo,
    onDiscardFile,
    onCreateBranch,
  }: {
    conversationId?: string
    browser: ProjectConversationWorkspaceBrowserState
    syncPrompt?: ProjectConversationWorkspaceSyncPrompt | null
    syncError?: string
    syncInFlight?: boolean
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    runtimeActive?: boolean
    terminalManager: ReturnType<typeof createTerminalManager>
    onRefreshWorkspace?: () => Promise<void> | void
    onSyncWorkspace?: () => Promise<void>
    onClose?: () => void
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onCreateBranchName?: (branchName: string) => Promise<void>
    onGitRemoteOp?: (op: ProjectConversationWorkspaceGitRemoteOp) => Promise<void>
    onStageFile?: (path: string) => Promise<void>
    onStageAll?: () => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onCommitRepo?: (message: string) => Promise<void>
    onDiscardFile?: (path: string) => Promise<void>
    onCreateBranch?: (commitId: string) => void
  } = $props()
  let pathCopied = $state(false)

  function copyWorkspacePath() {
    const path = browser.metadata?.workspacePath
    if (!path) return
    navigator.clipboard.writeText(path)
    pathCopied = true
    setTimeout(() => (pathCopied = false), 1500)
  }
</script>

<ProjectConversationWorkspaceBrowserToolbar
  workspacePath={browser.metadata?.workspacePath ?? ''}
  {pathCopied}
  showTerminalButton={Boolean(browser.metadata?.available)}
  terminalPanelOpen={terminalManager.panelOpen}
  {conversationId}
  metadataLoading={browser.metadataLoading}
  onCopyWorkspacePath={copyWorkspacePath}
  onToggleTerminal={() => terminalManager.togglePanel()}
  {onRefreshWorkspace}
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
    onSync={onSyncWorkspace}
  />
{:else}
  <div class="flex min-h-0 flex-1 flex-col">
    {#if syncPrompt}
      <ProjectConversationWorkspaceSyncBanner
        prompt={syncPrompt}
        {syncError}
        {syncInFlight}
        onSync={onSyncWorkspace}
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
      {onCheckoutBranch}
      {onCreateBranchName}
      {onGitRemoteOp}
      {onStageFile}
      {onStageAll}
      {onUnstage}
      {onCommitRepo}
      {onDiscardFile}
      {onCreateBranch}
    />
  </div>
{/if}
