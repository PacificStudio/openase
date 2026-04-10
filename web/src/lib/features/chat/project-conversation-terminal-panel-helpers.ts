import type { ProjectConversationTerminalSession } from '$lib/api/chat'

export type TerminalPanelStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error'
export type TerminalLaunchPreset = 'context' | 'workspace-root'
export type TerminalLaunchTarget = {
  label: string
  repoPath?: string
  cwdPath?: string
}

export type TerminalServerFrame =
  | { type: 'ready' }
  | { type: 'output'; data: Uint8Array }
  | { type: 'exit'; exitCode: number; signal?: string }
  | { type: 'error'; message: string }

export async function mountProjectConversationTerminal(input: {
  element: HTMLDivElement
  onData: (data: string) => void
  onResize: (size: { cols: number; rows: number }) => void
}) {
  const [{ Terminal }, { FitAddon }] = await Promise.all([
    import('@xterm/xterm'),
    import('@xterm/addon-fit'),
  ])
  const terminal = new Terminal({
    allowTransparency: true,
    convertEol: false,
    cursorBlink: true,
    fontFamily: '"JetBrains Mono", "SFMono-Regular", ui-monospace, monospace',
    fontSize: 12,
    scrollback: 5000,
    theme: {
      background: '#08131f',
      foreground: '#e6edf5',
      cursor: '#ffd36f',
      cursorAccent: '#08131f',
      selectionBackground: 'rgba(110, 168, 254, 0.3)',
    },
  })
  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(input.element)
  fitAddon.fit()

  const dataSubscription = terminal.onData(input.onData)
  const resizeSubscription = terminal.onResize(input.onResize)

  return {
    fitAddon,
    terminal,
    dispose() {
      dataSubscription.dispose()
      resizeSubscription.dispose()
      terminal.dispose()
    },
  }
}

export function buildExitMessage(exitCode: number, signal?: string) {
  if (signal && signal.trim() !== '') {
    return `Terminal closed with signal ${signal}.`
  }
  return `Terminal closed with exit code ${exitCode}.`
}

export function buildTerminalWebSocketURL(session: ProjectConversationTerminalSession) {
  const url = new URL(session.wsPath, window.location.origin)
  url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('attach_token', session.attachToken)
  return url.toString()
}

export function parseTerminalServerFrame(data: unknown): TerminalServerFrame {
  if (typeof data !== 'string') {
    throw new Error('Terminal websocket frame must be text.')
  }
  const raw = JSON.parse(data) as unknown
  if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) {
    throw new Error('Terminal websocket frame must be an object.')
  }
  const object = raw as Record<string, unknown>
  const type = readRequiredString(object, 'type')

  switch (type) {
    case 'ready':
      return { type }
    case 'output':
      return {
        type,
        data: decodeTerminalPayload(readRequiredString(object, 'data')),
      }
    case 'exit':
      return {
        type,
        exitCode: readRequiredNumber(object, 'exit_code'),
        signal: readOptionalString(object, 'signal'),
      }
    case 'error':
      return {
        type,
        message: readRequiredString(object, 'message'),
      }
    default:
      throw new Error(`Terminal websocket frame type ${type} is unsupported.`)
  }
}

export function encodeTerminalPayload(value: string) {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary)
}

export function resolveTerminalLaunchTarget(input: {
  preset: TerminalLaunchPreset
  workspacePath: string
  selectedRepoPath: string
  selectedFilePath: string
}): TerminalLaunchTarget {
  if (input.preset === 'workspace-root' || !input.selectedRepoPath) {
    return {
      label: input.workspacePath || 'workspace root',
    }
  }

  const cwdPath = resolveCurrentDirectory(input.selectedFilePath)
  return {
    label: cwdPath ? `${input.selectedRepoPath}/${cwdPath}` : input.selectedRepoPath,
    repoPath: input.selectedRepoPath,
    cwdPath: cwdPath || undefined,
  }
}

function decodeTerminalPayload(value: string) {
  const binary = atob(value)
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}

function readRequiredNumber(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Terminal websocket frame field ${key} must be a number.`)
  }
  return value
}

function readRequiredString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Terminal websocket frame field ${key} must be a non-empty string.`)
  }
  return value
}

function readOptionalString(object: Record<string, unknown>, key: string) {
  const value = object[key]
  return typeof value === 'string' && value.trim() !== '' ? value : undefined
}

function resolveCurrentDirectory(selectedFilePath: string) {
  if (selectedFilePath.trim() === '') {
    return ''
  }
  const lastSlashIndex = selectedFilePath.lastIndexOf('/')
  return lastSlashIndex <= 0 ? '' : selectedFilePath.slice(0, lastSlashIndex)
}
