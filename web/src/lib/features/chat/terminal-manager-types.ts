import type { ProjectConversationTerminalSession } from '$lib/api/chat'
import type { TerminalPanelStatus } from './project-conversation-terminal-panel-helpers'

export interface TerminalInstance {
  id: string
  label: string
  status: TerminalPanelStatus
  statusMessage: string
  sessionID: string
}

export type TerminalInstanceRuntime = {
  mountRevision: number
  connectRevision: number
  reconnectAttempts: number
  reconnectEnabled: boolean
  reconnectTimer: ReturnType<typeof setTimeout> | null
  session: ProjectConversationTerminalSession | null
}

export type MountedTerminal = {
  terminal: import('@xterm/xterm').Terminal
  fitAddon: import('@xterm/addon-fit').FitAddon
  dispose: () => void
}
