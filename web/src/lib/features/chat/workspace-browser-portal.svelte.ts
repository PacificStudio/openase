import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'

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
  /** File path to navigate to when the browser opens (consumed once). */
  pendingFilePath = $state('')

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

  /** Consume and clear the pending file path. */
  consumePendingFile(): string {
    const path = this.pendingFilePath
    this.pendingFilePath = ''
    return path
  }
}

export const workspaceBrowserPortal = new WorkspaceBrowserPortal()
