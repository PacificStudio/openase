import {
  encodeTerminalPayload,
  mountProjectConversationTerminal,
} from './project-conversation-terminal-panel-helpers'
import { createTerminalConnectionHelpers } from './terminal-manager-connection'
import type {
  MountedTerminal,
  TerminalInstance,
  TerminalInstanceRuntime,
} from './terminal-manager-types'

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
  const xtermMap = new Map<string, MountedTerminal>()
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
        session: null,
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

  const { attachSocket, matchesConnectionState, resolveTerminalSession, setConnectingStatus } =
    createTerminalConnectionHelpers({
      getConversationId: input.getConversationId,
      hasInstance,
      listInstances: () => instances,
      runtimeMap,
      socketMap,
      scheduleReconnect,
      updateInstance,
    })

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
    closeSocket(id, { updateStatus: false, reconnect: false, terminate: true })
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
      terminate: boolean
    },
  ) {
    const runtime = ensureRuntime(id)
    runtime.reconnectEnabled = options.reconnect
    clearReconnectTimer(id)

    const socket = socketMap.get(id)
    socketMap.delete(id)
    if (socket?.readyState === WebSocket.OPEN) {
      if (options.terminate) {
        socket.send(JSON.stringify({ type: 'close' }))
      }
      socket.close()
    } else if (socket?.readyState === WebSocket.CONNECTING) {
      socket.close()
    }
    if (options.terminate) {
      runtime.session = null
    }
    if (options.updateStatus) {
      updateInstance(id, { status: 'closed', statusMessage: 'Terminal closed.', sessionID: '' })
    }
  }

  function scheduleReconnect(id: string, label: string) {
    const runtime = runtimeMap.get(id)
    if (
      !runtime ||
      !runtime.reconnectEnabled ||
      !runtime.session ||
      !hasInstance(id) ||
      !xtermMap.has(id)
    ) {
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

    closeSocket(id, { updateStatus: false, reconnect: false, terminate: false })
    runtime.connectRevision += 1
    const connectRevision = runtime.connectRevision
    runtime.reconnectEnabled = true
    clearReconnectTimer(id)
    entry.fitAddon.fit()

    const label = workspacePath || 'workspace root'
    updateInstance(id, { label })
    setConnectingStatus(id, label, isReconnect)

    const session = await resolveTerminalSession({
      id,
      conversationId,
      connectRevision,
      runtime,
      terminal: entry.terminal,
    })
    if (!session) {
      return
    }

    const currentEntry = xtermMap.get(id)
    if (!currentEntry || !matchesConnectionState(id, conversationId, connectRevision)) {
      return
    }

    runtime.session = session
    updateInstance(id, { sessionID: session.id })
    attachSocket({
      id,
      session,
      connectRevision,
      terminal: currentEntry.terminal,
      runtime,
      label,
    })
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
