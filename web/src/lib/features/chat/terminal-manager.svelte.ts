/* eslint-disable max-lines, max-lines-per-function, complexity */
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
  type TerminalPanelStatus,
  type TerminalServerFrame,
} from './project-conversation-terminal-panel-helpers'

export interface TerminalInstance {
  id: string
  label: string
  status: TerminalPanelStatus
  statusMessage: string
  sessionID: string
}

type TerminalInstanceRuntime = {
  mountRevision: number
  connectRevision: number
  reconnectAttempts: number
  reconnectEnabled: boolean
  reconnectTimer: ReturnType<typeof setTimeout> | null
}

let nextId = 1
const reconnectDelaysMs = [750, 1_500, 3_000, 5_000] as const

function generateId(): string {
  return `term-${nextId++}`
}

export function createTerminalManager(input: {
  getConversationId: () => string
  getWorkspacePath: () => string
}) {
  let instances = $state<TerminalInstance[]>([])
  let activeId = $state<string>('')
  let panelOpen = $state(false)

  // Internal state per instance (not reactive, keyed by id)
  const xtermMap = new Map<
    string,
    {
      terminal: import('@xterm/xterm').Terminal
      fitAddon: import('@xterm/addon-fit').FitAddon
      dispose: () => void
    }
  >()
  const socketMap = new Map<string, WebSocket>()
  const elementMap = new Map<string, HTMLDivElement>()
  const resizeObserverMap = new Map<string, ResizeObserver>()
  const runtimeMap = new Map<string, TerminalInstanceRuntime>()

  function updateInstance(id: string, updates: Partial<TerminalInstance>) {
    instances = instances.map((inst) => (inst.id === id ? { ...inst, ...updates } : inst))
  }

  function getActiveInstance(): TerminalInstance | undefined {
    return instances.find((i) => i.id === activeId)
  }

  function hasInstance(id: string) {
    return instances.some((inst) => inst.id === id)
  }

  function ensureRuntime(id: string) {
    let runtime = runtimeMap.get(id)
    if (!runtime) {
      runtime = {
        mountRevision: 0,
        connectRevision: 0,
        reconnectAttempts: 0,
        reconnectEnabled: false,
        reconnectTimer: null,
      }
      runtimeMap.set(id, runtime)
    }
    return runtime
  }

  function clearReconnectTimer(id: string) {
    const runtime = runtimeMap.get(id)
    if (!runtime?.reconnectTimer) {
      return
    }
    clearTimeout(runtime.reconnectTimer)
    runtime.reconnectTimer = null
  }

  function forgetRuntime(id: string) {
    clearReconnectTimer(id)
    runtimeMap.delete(id)
  }

  function nextReconnectDelay(attempt: number) {
    return reconnectDelaysMs[Math.min(attempt - 1, reconnectDelaysMs.length - 1)]
  }

  async function mountTerminal(id: string, element: HTMLDivElement) {
    // Prevent double-mount
    if (xtermMap.has(id) && elementMap.get(id) === element) return

    const runtime = ensureRuntime(id)
    runtime.mountRevision += 1
    const mountRevision = runtime.mountRevision

    // Clean up previous mount for this id
    unmountTerminal(id, false)
    elementMap.set(id, element)

    const mounted = await mountProjectConversationTerminal({
      element,
      onData: (data) => {
        const socket = socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'input', data: encodeTerminalPayload(data) }))
      },
      onResize: ({ cols, rows }) => {
        const socket = socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'resize', cols, rows }))
      },
    })

    if (
      !hasInstance(id) ||
      runtimeMap.get(id)?.mountRevision !== mountRevision ||
      elementMap.get(id) !== element
    ) {
      mounted.dispose()
      return
    }

    xtermMap.set(id, mounted)

    const ro = new ResizeObserver(() => {
      const entry = xtermMap.get(id)
      if (!entry) return
      entry.fitAddon.fit()
      const socket = socketMap.get(id)
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({ type: 'resize', cols: entry.terminal.cols, rows: entry.terminal.rows }),
        )
      }
    })
    ro.observe(element)
    resizeObserverMap.set(id, ro)
  }

  function unmountTerminal(id: string, forget: boolean) {
    resizeObserverMap.get(id)?.disconnect()
    resizeObserverMap.delete(id)
    closeSocket(id, { updateStatus: false, reconnect: false })
    xtermMap.get(id)?.dispose()
    xtermMap.delete(id)
    elementMap.delete(id)
    if (forget) {
      forgetRuntime(id)
    }
  }

  function closeSocket(
    id: string,
    options: {
      updateStatus: boolean
      reconnect: boolean
    },
  ) {
    const runtime = ensureRuntime(id)
    runtime.reconnectEnabled = options.reconnect
    clearReconnectTimer(id)

    const socket = socketMap.get(id)
    socketMap.delete(id)
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'close' }))
      socket.close()
    } else if (socket?.readyState === WebSocket.CONNECTING) {
      socket.close()
    }
    if (options.updateStatus) {
      updateInstance(id, { status: 'closed', statusMessage: 'Terminal closed.', sessionID: '' })
    }
  }

  function scheduleReconnect(id: string, label: string) {
    const runtime = runtimeMap.get(id)
    if (!runtime || !runtime.reconnectEnabled || !hasInstance(id) || !xtermMap.has(id)) {
      return
    }

    if (runtime.reconnectAttempts >= reconnectDelaysMs.length) {
      runtime.reconnectEnabled = false
      updateInstance(id, {
        status: 'error',
        statusMessage: 'Terminal disconnected. Reconnect attempts exhausted.',
        sessionID: '',
      })
      return
    }

    runtime.reconnectAttempts += 1
    const delay = nextReconnectDelay(runtime.reconnectAttempts)
    updateInstance(id, {
      status: 'connecting',
      statusMessage: `Reconnecting shell in ${label}...`,
      sessionID: '',
    })
    runtime.reconnectTimer = setTimeout(() => {
      runtime.reconnectTimer = null
      if (!runtime.reconnectEnabled || !hasInstance(id) || !xtermMap.has(id)) {
        return
      }
      void connectTerminal(id, true)
    }, delay)
  }

  async function connectTerminal(id: string, isReconnect = false) {
    const conversationId = input.getConversationId()
    const workspacePath = input.getWorkspacePath()
    const runtime = ensureRuntime(id)
    const entry = xtermMap.get(id)
    if (!conversationId || !entry || !hasInstance(id)) return

    closeSocket(id, { updateStatus: false, reconnect: false })
    runtime.connectRevision += 1
    const connectRevision = runtime.connectRevision
    runtime.reconnectEnabled = true
    clearReconnectTimer(id)
    entry.terminal.reset()
    entry.fitAddon.fit()

    const label = workspacePath || 'workspace root'
    updateInstance(id, {
      status: 'connecting',
      statusMessage: isReconnect ? `Reconnecting shell in ${label}...` : `Starting shell in ${label}...`,
      label,
    })

    let session: ProjectConversationTerminalSession
    try {
      const payload = await createProjectConversationTerminalSession(conversationId, {
        mode: 'shell',
        cols: entry.terminal.cols > 0 ? entry.terminal.cols : 120,
        rows: entry.terminal.rows > 0 ? entry.terminal.rows : 32,
      })
      session = payload.terminalSession
    } catch (error) {
      if (
        !hasInstance(id) ||
        input.getConversationId() !== conversationId ||
        runtimeMap.get(id)?.connectRevision !== connectRevision
      ) {
        return
      }
      updateInstance(id, {
        status: 'error',
        statusMessage: error instanceof Error ? error.message : 'Failed to create terminal session.',
      })
      return
    }

    const currentEntry = xtermMap.get(id)
    if (
      !hasInstance(id) ||
      input.getConversationId() !== conversationId ||
      runtimeMap.get(id)?.connectRevision !== connectRevision ||
      !currentEntry
    ) {
      return
    }

    updateInstance(id, { sessionID: session.id })
    const socket = new WebSocket(buildTerminalWebSocketURL(session))
    socketMap.set(id, socket)

    socket.onmessage = (event) => {
      if (
        socketMap.get(id) !== socket ||
        runtimeMap.get(id)?.connectRevision !== connectRevision ||
        !hasInstance(id)
      ) {
        return
      }
      try {
        handleFrame(id, parseTerminalServerFrame(event.data), socket, currentEntry.terminal)
      } catch (error) {
        runtime.reconnectEnabled = false
        updateInstance(id, {
          status: 'error',
          statusMessage: error instanceof Error ? error.message : 'Failed to parse terminal output.',
        })
        socket.close()
      }
    }

    socket.onerror = () => {
      if (
        socketMap.get(id) !== socket ||
        runtimeMap.get(id)?.connectRevision !== connectRevision ||
        !hasInstance(id)
      ) {
        return
      }
      updateInstance(id, { status: 'connecting', statusMessage: `Reconnecting shell in ${label}...` })
    }

    socket.onclose = () => {
      if (
        socketMap.get(id) !== socket ||
        runtimeMap.get(id)?.connectRevision !== connectRevision ||
        !hasInstance(id)
      ) {
        return
      }
      socketMap.delete(id)
      const inst = instances.find((i) => i.id === id)
      if (!inst) {
        return
      }
      if (runtime.reconnectEnabled && (inst.status === 'connecting' || inst.status === 'open')) {
        scheduleReconnect(id, inst.label)
        return
      }
      if (inst.status === 'connecting' || inst.status === 'open') {
        updateInstance(id, {
          status: 'closed',
          statusMessage: 'Terminal disconnected.',
          sessionID: '',
        })
      }
    }
  }

  function handleFrame(
    id: string,
    frame: TerminalServerFrame,
    socket: WebSocket,
    xterm: import('@xterm/xterm').Terminal,
  ) {
    const inst = instances.find((i) => i.id === id)
    const runtime = ensureRuntime(id)
    switch (frame.type) {
      case 'ready':
        runtime.reconnectAttempts = 0
        updateInstance(id, { status: 'open', statusMessage: `Shell attached to ${inst?.label}.` })
        xterm.focus()
        return
      case 'output':
        xterm.write(frame.data)
        return
      case 'exit':
        runtime.reconnectEnabled = false
        updateInstance(id, {
          status: 'closed',
          statusMessage: buildExitMessage(frame.exitCode, frame.signal),
          sessionID: '',
        })
        socket.close()
        return
      case 'error':
        runtime.reconnectEnabled = false
        updateInstance(id, { status: 'error', statusMessage: frame.message, sessionID: '' })
        socket.close()
        return
    }
  }

  function createInstance(): string {
    const id = generateId()
    const index = instances.length + 1
    instances = [
      ...instances,
      {
        id,
        label: `Terminal ${index}`,
        status: 'idle',
        statusMessage: 'Connecting...',
        sessionID: '',
      },
    ]
    activeId = id
    return id
  }

  function removeInstance(id: string) {
    const closingIndex = instances.findIndex((inst) => inst.id === id)
    unmountTerminal(id, true)
    instances = instances.filter((i) => i.id !== id)
    if (activeId === id) {
      const nextActive = instances[closingIndex] ?? instances[Math.max(closingIndex - 1, 0)]
      activeId = nextActive?.id ?? ''
    }
    if (instances.length === 0) {
      panelOpen = false
    }
  }

  function openPanel() {
    panelOpen = true
    if (instances.length === 0) {
      createInstance()
    }
  }

  function togglePanel() {
    if (panelOpen) {
      panelOpen = false
    } else {
      openPanel()
    }
  }

  function closePanel() {
    panelOpen = false
  }

  function disposeAll() {
    for (const inst of instances) {
      unmountTerminal(inst.id, true)
    }
    instances = []
    activeId = ''
    panelOpen = false
  }

  /** Refits all visible terminals (call after panel resize). */
  function refitAll() {
    for (const [, entry] of xtermMap) {
      entry.fitAddon.fit()
    }
  }

  return {
    get instances() {
      return instances
    },
    get activeId() {
      return activeId
    },
    set activeId(id: string) {
      activeId = id
    },
    get panelOpen() {
      return panelOpen
    },
    getActiveInstance,
    mountTerminal,
    connectTerminal,
    createInstance,
    removeInstance,
    openPanel,
    togglePanel,
    closePanel,
    disposeAll,
    refitAll,
  }
}

export type TerminalManager = ReturnType<typeof createTerminalManager>
