import type { TerminalInstanceRuntime } from './terminal-manager-types'

const reconnectDelaysMs = [750, 1_500, 3_000, 5_000] as const
export const TERMINAL_RECONNECT_ATTEMPT_LIMIT = reconnectDelaysMs.length

let nextTerminalID = 1

export function generateTerminalManagerID(): string {
  return `term-${nextTerminalID++}`
}

export function ensureTerminalRuntime(
  runtimeMap: Map<string, TerminalInstanceRuntime>,
  id: string,
) {
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

export function clearTerminalReconnectTimer(
  runtimeMap: Map<string, TerminalInstanceRuntime>,
  id: string,
) {
  const runtime = runtimeMap.get(id)
  if (!runtime?.reconnectTimer) {
    return
  }
  clearTimeout(runtime.reconnectTimer)
  runtime.reconnectTimer = null
}

export function forgetTerminalRuntime(
  runtimeMap: Map<string, TerminalInstanceRuntime>,
  id: string,
) {
  clearTerminalReconnectTimer(runtimeMap, id)
  runtimeMap.delete(id)
}

export function nextTerminalReconnectDelay(attempt: number) {
  return reconnectDelaysMs[Math.min(attempt - 1, reconnectDelaysMs.length - 1)]
}
