import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
import type { ChatDiffPayload } from '$lib/api/chat'

/**
 * Shared reactive state that lets the workspace browser render outside the
 * assistant panel (in the main content area) while the conversation controller
 * that owns the data lives inside the panel.
 *
 * - Written by: ProjectConversationContent (syncs conversation data)
 * - Read by: ProjectShellFrame (renders the browser in <main>)
 */
class WorkspaceBrowserPortal {
  open = $state(false)
  conversationId = $state('')
  workspaceDiff: ProjectConversationWorkspaceDiff | null = $state(null)
  workspaceDiffLoading = $state(false)
  runtimeActive = $state(false)
  syncGeneration = $state(0)
  onSyncWorkspace: null | (() => Promise<void> | void) = null
  /** File path to navigate to when the browser opens (consumed once). */
  pendingFilePath = $state('')
  pendingPatch: { diff: ChatDiffPayload; autoApply: boolean } | null = $state(null)

  toggle() {
    this.open = !this.open
  }

  close() {
    this.open = false
  }

  /** Open the browser and navigate to a specific file. */
  openToFile(filePath: string) {
    this.pendingFilePath = filePath
    this.open = true
  }

  reviewPatch(diff: ChatDiffPayload, autoApply = false) {
    this.pendingPatch = { diff, autoApply }
    this.openToFile(diff.file)
  }

  /** Consume and clear the pending file path. */
  consumePendingFile(): string {
    const path = this.pendingFilePath
    this.pendingFilePath = ''
    return path
  }

  consumePendingPatch() {
    const patch = this.pendingPatch
    this.pendingPatch = null
    return patch
  }

  markWorkspaceSynced() {
    this.syncGeneration += 1
  }
}

export const workspaceBrowserPortal = new WorkspaceBrowserPortal()
