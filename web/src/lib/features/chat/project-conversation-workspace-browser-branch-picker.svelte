<script lang="ts">
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
    ProjectConversationWorkspaceDiffRepo,
  } from '$lib/api/chat'
  import WorkspaceBrowserBranchPopover from './workspace-browser-branch-popover.svelte'
  import WorkspaceBrowserChangesPanel from './workspace-browser-changes-panel.svelte'

  let {
    currentRef = null,
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    selectedRepo = null,
    selectedRepoDiff = null,
    selectedFilePath = '',
    onCheckoutBranch,
    onCreateBranchName,
    onStageFile,
    onStageAll,
    onUnstage,
    onCommitRepo,
    onDiscardFile,
    onSelectFile,
  }: {
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    selectedRepo?: { branch: string; headCommit: string } | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    selectedFilePath?: string
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onCreateBranchName?: (branchName: string) => Promise<void>
    onStageFile?: (path: string) => Promise<void>
    onStageAll?: () => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onCommitRepo?: (message: string) => Promise<void>
    onDiscardFile?: (path: string) => Promise<void>
    onSelectFile?: (path: string) => void
  } = $props()
</script>

{#if selectedRepo || currentRef}
  <div class="border-border shrink-0 border-t">
    <div class="bg-muted/30 flex items-center text-[11px]">
      <WorkspaceBrowserBranchPopover
        {currentRef}
        {localBranches}
        {remoteBranches}
        {repoRefsLoading}
        {repoRefsError}
        {checkoutBlockers}
        {onCheckoutBranch}
        {onCreateBranchName}
      />
      <WorkspaceBrowserChangesPanel
        commitHash={selectedRepo?.headCommit ?? ''}
        {selectedRepoDiff}
        {selectedFilePath}
        {onStageFile}
        {onStageAll}
        {onUnstage}
        {onCommitRepo}
        {onDiscardFile}
        {onSelectFile}
      />
    </div>
  </div>
{/if}
