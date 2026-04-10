<script lang="ts">
  import '@xterm/xterm/css/xterm.css'
  import { onDestroy, onMount } from 'svelte'
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import {
    AlertCircle,
    LoaderCircle,
    RefreshCcw,
    SquareTerminal,
    Terminal,
    X,
  } from '@lucide/svelte'
  import {
    createProjectConversationTerminalSession,
    type ProjectConversationTerminalSession,
  } from '$lib/api/chat'

  type TerminalPanelStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error'
  type TerminalLaunchPreset = 'context' | 'workspace-root'
  type TerminalLaunchTarget = {
    label: string
    repoPath?: string
    cwdPath?: string
  }
  type TerminalServerFrame =
    | { type: 'ready' }
    | { type: 'output'; data: Uint8Array }
    | { type: 'exit'; exitCode: number; signal?: string }
    | { type: 'error'; message: string }

  let {
    conversationId = '',
    workspacePath = '',
    selectedRepoPath = '',
    currentTreePath = '',
    selectedFilePath = '',
  }: {
    conversationId?: string
    workspacePath?: string
    selectedRepoPath?: string
    currentTreePath?: string
    selectedFilePath?: string
  } = $props()

  let terminalElement: HTMLDivElement | null = null
  let xterm: import('@xterm/xterm').Terminal | null = null
  let fitAddon: import('@xterm/addon-fit').FitAddon | null = null
  let resizeObserver: ResizeObserver | null = null
  let terminalReady = $state(false)
  let status = $state<TerminalPanelStatus>('idle')
  let statusMessage = $state('Open a shell in the selected repo or at the workspace root.')
  let lastLaunchLabel = $state('')
  let sessionID = $state('')
  let activeSocket: WebSocket | null = null
  let lastConversationId = $state('')

  const contextTarget = $derived(
    resolveTerminalLaunchTarget({
      preset: 'context',
      workspacePath,
      selectedRepoPath,
      currentTreePath,
      selectedFilePath,
    }),
  )
  const workspaceRootTarget = $derived(
    resolveTerminalLaunchTarget({
      preset: 'workspace-root',
      workspacePath,
      selectedRepoPath,
      currentTreePath,
      selectedFilePath,
    }),
  )

  onMount(() => {
    let mounted = true
    let dispose = () => {}

    void (async () => {
      const [{ Terminal: XTerm }, { FitAddon }] = await Promise.all([
        import('@xterm/xterm'),
        import('@xterm/addon-fit'),
      ])
      if (!mounted || !terminalElement) {
        return
      }

      const terminal = new XTerm({
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
      const fit = new FitAddon()
      terminal.loadAddon(fit)
      terminal.open(terminalElement)
      fit.fit()

      const dataSubscription = terminal.onData((data) => {
        if (activeSocket?.readyState !== WebSocket.OPEN) {
          return
        }
        activeSocket.send(
          JSON.stringify({
            type: 'input',
            data: encodeTerminalPayload(data),
          }),
        )
      })
      const resizeSubscription = terminal.onResize(({ cols, rows }) => {
        if (activeSocket?.readyState !== WebSocket.OPEN) {
          return
        }
        activeSocket.send(JSON.stringify({ type: 'resize', cols, rows }))
      })

      resizeObserver = new ResizeObserver(() => {
        if (!xterm || !fitAddon) {
          return
        }
        fitAddon.fit()
        if (activeSocket?.readyState === WebSocket.OPEN) {
          activeSocket.send(JSON.stringify({ type: 'resize', cols: xterm.cols, rows: xterm.rows }))
        }
      })
      resizeObserver.observe(terminalElement)

      xterm = terminal
      fitAddon = fit
      terminalReady = true

      dispose = () => {
        dataSubscription.dispose()
        resizeSubscription.dispose()
        resizeObserver?.disconnect()
        resizeObserver = null
        closeTerminal({ updateStatus: false })
        terminal.dispose()
        xterm = null
        fitAddon = null
        terminalReady = false
      }
    })()

    return () => {
      mounted = false
      dispose()
    }
  })

  onDestroy(() => {
    closeTerminal({ updateStatus: false })
  })

  $effect(() => {
    if (!conversationId) {
      lastConversationId = ''
      closeTerminal({ updateStatus: false })
      resetTerminalState()
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
    resetTerminalState()
    xterm?.clear()
  })

  function resetTerminalState() {
    status = 'idle'
    sessionID = ''
    lastLaunchLabel = ''
    statusMessage = 'Open a shell in the selected repo or at the workspace root.'
  }

  function closeTerminal(options: { updateStatus: boolean }) {
    const socket = activeSocket
    activeSocket = null
    sessionID = ''

    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'close' }))
      socket.close()
    } else if (socket && socket.readyState === WebSocket.CONNECTING) {
      socket.close()
    }

    if (options.updateStatus && (status === 'connecting' || status === 'open')) {
      status = 'closed'
      statusMessage = 'Terminal closed.'
    }
  }

  async function openTerminal(preset: TerminalLaunchPreset) {
    if (!conversationId || !terminalReady || !xterm || !fitAddon) {
      return
    }

    const target = preset === 'workspace-root' ? workspaceRootTarget : contextTarget
    closeTerminal({ updateStatus: false })
    xterm.reset()
    fitAddon.fit()

    const cols = xterm.cols > 0 ? xterm.cols : 120
    const rows = xterm.rows > 0 ? xterm.rows : 32
    status = 'connecting'
    statusMessage = `Starting shell in ${target.label}…`
    lastLaunchLabel = target.label

    let session: ProjectConversationTerminalSession
    try {
      const payload = await createProjectConversationTerminalSession(conversationId, {
        mode: 'shell',
        repoPath: target.repoPath,
        cwdPath: target.cwdPath,
        cols,
        rows,
      })
      session = payload.terminalSession
    } catch (error) {
      status = 'error'
      statusMessage = error instanceof Error ? error.message : 'Failed to create terminal session.'
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
        const frame = parseTerminalServerFrame(event.data)
        handleServerFrame(frame, socket)
      } catch (error) {
        status = 'error'
        statusMessage = error instanceof Error ? error.message : 'Failed to parse terminal output.'
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

  function buildExitMessage(exitCode: number, signal?: string) {
    if (signal && signal.trim() !== '') {
      return `Terminal closed with signal ${signal}.`
    }
    return `Terminal closed with exit code ${exitCode}.`
  }

  function buildTerminalWebSocketURL(session: ProjectConversationTerminalSession) {
    const url = new URL(session.wsPath, window.location.origin)
    url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    url.searchParams.set('attach_token', session.attachToken)
    return url.toString()
  }

  function parseTerminalServerFrame(data: unknown): TerminalServerFrame {
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

  function readRequiredNumber(object: Record<string, unknown>, key: string) {
    const value = object[key]
    if (typeof value !== 'number' || Number.isNaN(value)) {
      throw new Error(`Terminal websocket frame field ${key} must be a number.`)
    }
    return value
  }

  function encodeTerminalPayload(value: string) {
    const bytes = new TextEncoder().encode(value)
    let binary = ''
    for (const byte of bytes) {
      binary += String.fromCharCode(byte)
    }
    return btoa(binary)
  }

  function decodeTerminalPayload(value: string) {
    const binary = atob(value)
    return Uint8Array.from(binary, (character) => character.charCodeAt(0))
  }

  function resolveTerminalLaunchTarget(input: {
    preset: TerminalLaunchPreset
    workspacePath: string
    selectedRepoPath: string
    currentTreePath: string
    selectedFilePath: string
  }): TerminalLaunchTarget {
    if (input.preset === 'workspace-root' || !input.selectedRepoPath) {
      return {
        label: input.workspacePath || 'workspace root',
      }
    }

    const cwdPath = resolveCurrentDirectory(input.currentTreePath, input.selectedFilePath)
    return {
      label: cwdPath ? `${input.selectedRepoPath}/${cwdPath}` : input.selectedRepoPath,
      repoPath: input.selectedRepoPath,
      cwdPath: cwdPath || undefined,
    }
  }

  function resolveCurrentDirectory(currentTreePath: string, selectedFilePath: string) {
    if (selectedFilePath.trim() === '') {
      return currentTreePath.trim()
    }
    const lastSlashIndex = selectedFilePath.lastIndexOf('/')
    return lastSlashIndex <= 0 ? '' : selectedFilePath.slice(0, lastSlashIndex)
  }
</script>

<div class="flex min-h-0 flex-1 flex-col">
  <div
    class="border-border bg-muted/20 flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3"
  >
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <SquareTerminal class="text-muted-foreground size-4 shrink-0" />
        <p class="text-sm font-semibold">Shell terminal</p>
      </div>
      <p class="text-muted-foreground truncate text-[11px]">
        {lastLaunchLabel || contextTarget.label}
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        disabled={!conversationId || !terminalReady}
        onclick={() => void openTerminal('context')}
      >
        {#if status === 'connecting' && sessionID}
          <LoaderCircle class="mr-1.5 size-3.5 animate-spin" />
        {:else}
          <Terminal class="mr-1.5 size-3.5" />
        {/if}
        Open here
      </Button>
      <Button
        variant="ghost"
        size="sm"
        disabled={!conversationId || !terminalReady}
        onclick={() => void openTerminal('workspace-root')}
      >
        <RefreshCcw class="mr-1.5 size-3.5" />
        Workspace root
      </Button>
      <Button
        variant="ghost"
        size="sm"
        disabled={!sessionID}
        onclick={() => closeTerminal({ updateStatus: true })}
      >
        <X class="mr-1.5 size-3.5" />
        Close
      </Button>
    </div>
  </div>

  <div class="relative flex min-h-0 flex-1 flex-col bg-[#08131f]">
    <div
      bind:this={terminalElement}
      class="min-h-0 flex-1 px-3 py-3"
      data-testid="project-conversation-terminal"
    ></div>

    <div class="pointer-events-none absolute inset-x-4 bottom-4 flex justify-end">
      <div
        class={cn(
          'border-border/70 bg-background/90 pointer-events-auto max-w-md rounded-lg border px-3 py-2 shadow-sm backdrop-blur',
          status === 'error' && 'border-destructive/40',
        )}
      >
        <div class="flex items-start gap-2">
          {#if status === 'error'}
            <AlertCircle class="text-destructive mt-0.5 size-4 shrink-0" />
          {:else if status === 'connecting'}
            <LoaderCircle class="text-muted-foreground mt-0.5 size-4 shrink-0 animate-spin" />
          {:else}
            <SquareTerminal class="text-muted-foreground mt-0.5 size-4 shrink-0" />
          {/if}
          <div class="min-w-0">
            <p class="text-xs font-medium capitalize">{status}</p>
            <p class="text-muted-foreground text-xs">{statusMessage}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
