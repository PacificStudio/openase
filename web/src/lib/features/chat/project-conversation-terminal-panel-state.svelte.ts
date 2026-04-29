import {
  createProjectConversationTerminalSession,
  type ProjectConversationTerminalSession,
} from '$lib/api/chat'
import {
  buildExitMessage,
  buildTerminalWebSocketURL,
  encodeTerminalPayload,
  mountProjectConversationTerminal,
  parseTerminalServerFrame,
  resolveTerminalLaunchTarget,
  type TerminalLaunchPreset,
  type TerminalPanelStatus,
  type TerminalServerFrame,
} from './project-conversation-terminal-panel-helpers'
import { chatT } from './i18n'

export function createProjectConversationTerminalPanelState(input: {
  getConversationId: () => string
  getWorkspacePath: () => string
  getSelectedRepoPath: () => string
  getSelectedFilePath: () => string
}) {
  let xterm: import('@xterm/xterm').Terminal | null = null
  let fitAddon: import('@xterm/addon-fit').FitAddon | null = null
  let resizeObserver: ResizeObserver | null = null
  let terminalReady = $state(false)
  let status = $state<TerminalPanelStatus>('idle')
  let statusMessage = $state('Open a shell in the selected repo or at the workspace root.')
  let lastLaunchLabel = $state('')
  let sessionID = $state('')
  let lastConversationId = $state('')
  let activeSocket: WebSocket | null = null
  let mountedElement: HTMLDivElement | null = null
  let mountedDispose: (() => void) | null = null

  const contextTarget = $derived(
    resolveTerminalLaunchTarget({
      preset: 'context',
      workspacePath: input.getWorkspacePath(),
      selectedRepoPath: input.getSelectedRepoPath(),
      selectedFilePath: input.getSelectedFilePath(),
    }),
  )
  const workspaceRootTarget = $derived(
    resolveTerminalLaunchTarget({
      preset: 'workspace-root',
      workspacePath: input.getWorkspacePath(),
      selectedRepoPath: input.getSelectedRepoPath(),
      selectedFilePath: input.getSelectedFilePath(),
    }),
  )

  async function mount(element: HTMLDivElement) {
    if (mountedElement === element && mountedDispose) {
      return
    }

    dispose()
    mountedElement = element

    const mountedTerminal = await mountProjectConversationTerminal({
      element,
      onData: (data) => {
        if (activeSocket?.readyState !== WebSocket.OPEN) {
          return
        }
        activeSocket.send(
          JSON.stringify({
            type: 'input',
            data: encodeTerminalPayload(data),
          }),
        )
      },
      onResize: ({ cols, rows }) => {
        if (activeSocket?.readyState !== WebSocket.OPEN) {
          return
        }
        activeSocket.send(JSON.stringify({ type: 'resize', cols, rows }))
      },
    })

    if (mountedElement !== element) {
      mountedTerminal.dispose()
      return
    }

    xterm = mountedTerminal.terminal
    fitAddon = mountedTerminal.fitAddon
    terminalReady = true

    resizeObserver = new ResizeObserver(() => {
      if (!xterm || !fitAddon) {
        return
      }
      fitAddon.fit()
      if (activeSocket?.readyState === WebSocket.OPEN) {
        activeSocket.send(JSON.stringify({ type: 'resize', cols: xterm.cols, rows: xterm.rows }))
      }
    })
    resizeObserver.observe(element)

    mountedDispose = () => {
      resizeObserver?.disconnect()
      resizeObserver = null
      closeTerminal({ updateStatus: false })
      mountedTerminal.dispose()
      xterm = null
      fitAddon = null
      terminalReady = false
      mountedElement = null
      mountedDispose = null
    }
  }

  function dispose() {
    mountedDispose?.()
  }

  function syncConversation() {
    const conversationId = input.getConversationId()
    if (!conversationId) {
      lastConversationId = ''
      closeTerminal({ updateStatus: false })
      reset()
      return
    }
    if (!lastConversationId) {
      lastConversationId = conversationId
      return
    }
    if (lastConversationId === conversationId) {
      return
    }
    lastConversationId = conversationId
    closeTerminal({ updateStatus: false })
    reset()
    xterm?.clear()
  }

  function reset() {
    status = 'idle'
    sessionID = ''
    lastLaunchLabel = ''
    statusMessage = 'Open a shell in the selected repo or at the workspace root.'
  }

  function closeTerminal(options: { updateStatus: boolean }) {
    const socket = activeSocket
    activeSocket = null
    sessionID = ''

    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'close' }))
      socket.close()
    } else if (socket?.readyState === WebSocket.CONNECTING) {
      socket.close()
    }

    if (options.updateStatus && (status === 'connecting' || status === 'open')) {
      status = 'closed'
      statusMessage = 'Terminal closed.'
    }
  }

  async function openTerminal(preset: TerminalLaunchPreset) {
    const conversationId = input.getConversationId()
    if (!conversationId || !terminalReady || !xterm || !fitAddon) {
      return
    }

    const target = preset === 'workspace-root' ? workspaceRootTarget : contextTarget
    closeTerminal({ updateStatus: false })
    xterm.reset()
    fitAddon.fit()

    status = 'connecting'
    statusMessage = `Starting shell in ${target.label}...`
    lastLaunchLabel = target.label

    let session: ProjectConversationTerminalSession
    try {
      const payload = await createProjectConversationTerminalSession(conversationId, {
        mode: 'shell',
        repoPath: target.repoPath,
        cwdPath: target.cwdPath,
        cols: xterm.cols > 0 ? xterm.cols : 120,
        rows: xterm.rows > 0 ? xterm.rows : 32,
      })
      session = payload.terminalSession
    } catch (error) {
      status = 'error'
      statusMessage = error instanceof Error ? error.message : chatT('chat.terminal.errors.create')
      return
    }

    sessionID = session.id
    const socket = new WebSocket(buildTerminalWebSocketURL(session))
    activeSocket = socket

    socket.onmessage = (event) => {
      if (activeSocket !== socket) {
        return
      }
      try {
        handleServerFrame(parseTerminalServerFrame(event.data), socket)
      } catch (error) {
        status = 'error'
        statusMessage = error instanceof Error ? error.message : chatT('chat.terminal.errors.parse')
        socket.close()
      }
    }

    socket.onerror = () => {
      if (activeSocket !== socket) {
        return
      }
      status = 'error'
      statusMessage = 'Terminal connection failed.'
    }

    socket.onclose = () => {
      if (activeSocket !== socket) {
        return
      }
      activeSocket = null
      sessionID = ''
      if (status === 'connecting' || status === 'open') {
        status = 'closed'
        statusMessage =
          lastLaunchLabel.trim() !== ''
            ? `Terminal disconnected from ${lastLaunchLabel}.`
            : 'Terminal disconnected.'
      }
    }
  }

  function handleServerFrame(frame: TerminalServerFrame, socket: WebSocket) {
    switch (frame.type) {
      case 'ready':
        status = 'open'
        statusMessage = `Shell attached to ${lastLaunchLabel}.`
        xterm?.focus()
        return
      case 'output':
        xterm?.write(frame.data)
        return
      case 'exit':
        status = 'closed'
        statusMessage = buildExitMessage(frame.exitCode, frame.signal)
        sessionID = ''
        socket.close()
        return
      case 'error':
        status = 'error'
        statusMessage = frame.message
        sessionID = ''
        socket.close()
        return
    }
  }

  return {
    get contextTarget() {
      return contextTarget
    },
    get lastLaunchLabel() {
      return lastLaunchLabel
    },
    get sessionID() {
      return sessionID
    },
    get status() {
      return status
    },
    get statusMessage() {
      return statusMessage
    },
    get terminalReady() {
      return terminalReady
    },
    mount,
    dispose,
    syncConversation,
    closeTerminal,
    openTerminal,
  }
}
