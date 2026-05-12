import { createTerminalConnectionHelpers } from './terminal-manager-connection'
import {
  clearTerminalReconnectTimer,
  ensureTerminalRuntime,
  generateTerminalManagerID,
} from './terminal-manager-runtime'
import { createTerminalLifecycleHelpers } from './terminal-manager-lifecycle'
import type {
  MountedTerminal,
  TerminalInstance,
  TerminalInstanceRuntime,
} from './terminal-manager-types'

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

  const { closeSocket, mountTerminal, scheduleReconnect, unmountTerminal } =
    createTerminalLifecycleHelpers({
      runtimeMap,
      socketMap,
      xtermMap,
      elementMap,
      resizeObserverMap,
      hasInstance,
      updateInstance,
      connectTerminal,
    })

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

  async function connectTerminal(id: string, isReconnect = false) {
    const conversationId = input.getConversationId()
    const workspacePath = input.getWorkspacePath()
    const runtime = ensureTerminalRuntime(runtimeMap, id)
    const entry = xtermMap.get(id)
    if (!conversationId || !entry || !hasInstance(id)) return

    closeSocket(id, { updateStatus: false, reconnect: false, terminate: false })
    runtime.connectRevision += 1
    const connectRevision = runtime.connectRevision
    runtime.reconnectEnabled = true
    clearTerminalReconnectTimer(runtimeMap, id)
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
    const id = generateTerminalManagerID()
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
