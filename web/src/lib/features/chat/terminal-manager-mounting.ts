import {
  encodeTerminalPayload,
  mountProjectConversationTerminal,
} from './project-conversation-terminal-panel-helpers'
import { ensureTerminalRuntime, forgetTerminalRuntime } from './terminal-manager-runtime'
import type { MountedTerminal, TerminalInstanceRuntime } from './terminal-manager-types'

type CloseSocketOptions = {
  updateStatus: boolean
  reconnect: boolean
  terminate: boolean
}

type TerminalMountMaps = {
  xtermMap: Map<string, MountedTerminal>
  socketMap: Map<string, WebSocket>
  elementMap: Map<string, HTMLDivElement>
  resizeObserverMap: Map<string, ResizeObserver>
  runtimeMap: Map<string, TerminalInstanceRuntime>
}

export async function mountTerminalInstance(input: {
  id: string
  element: HTMLDivElement
  hasInstance: (id: string) => boolean
  closeSocket: (id: string, options: CloseSocketOptions) => void
  maps: TerminalMountMaps
}) {
  if (input.maps.xtermMap.has(input.id) && input.maps.elementMap.get(input.id) === input.element) {
    return
  }

  const runtime = ensureTerminalRuntime(input.maps.runtimeMap, input.id)
  runtime.mountRevision += 1
  const mountRevision = runtime.mountRevision

  unmountTerminalInstance({
    id: input.id,
    forget: false,
    closeSocket: input.closeSocket,
    maps: input.maps,
  })
  input.maps.elementMap.set(input.id, input.element)

  const mounted = await mountProjectConversationTerminal({
    element: input.element,
    onData: (data) => {
      const socket = input.maps.socketMap.get(input.id)
      if (socket?.readyState !== WebSocket.OPEN) return
      socket.send(JSON.stringify({ type: 'input', data: encodeTerminalPayload(data) }))
    },
    onResize: ({ cols, rows }) => {
      const socket = input.maps.socketMap.get(input.id)
      if (socket?.readyState !== WebSocket.OPEN) return
      socket.send(JSON.stringify({ type: 'resize', cols, rows }))
    },
  })

  if (
    !input.hasInstance(input.id) ||
    input.maps.runtimeMap.get(input.id)?.mountRevision !== mountRevision ||
    input.maps.elementMap.get(input.id) !== input.element
  ) {
    mounted.dispose()
    return
  }

  input.maps.xtermMap.set(input.id, mounted)

  const resizeObserver = new ResizeObserver(() => {
    const entry = input.maps.xtermMap.get(input.id)
    if (!entry) return
    entry.fitAddon.fit()
    const socket = input.maps.socketMap.get(input.id)
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(
        JSON.stringify({ type: 'resize', cols: entry.terminal.cols, rows: entry.terminal.rows }),
      )
    }
  })
  resizeObserver.observe(input.element)
  input.maps.resizeObserverMap.set(input.id, resizeObserver)
}

export function unmountTerminalInstance(input: {
  id: string
  forget: boolean
  closeSocket: (id: string, options: CloseSocketOptions) => void
  maps: TerminalMountMaps
}) {
  input.maps.resizeObserverMap.get(input.id)?.disconnect()
  input.maps.resizeObserverMap.delete(input.id)
  input.closeSocket(input.id, { updateStatus: false, reconnect: false, terminate: true })
  input.maps.xtermMap.get(input.id)?.dispose()
  input.maps.xtermMap.delete(input.id)
  input.maps.elementMap.delete(input.id)
  if (input.forget) {
    forgetTerminalRuntime(input.maps.runtimeMap, input.id)
  }
}
