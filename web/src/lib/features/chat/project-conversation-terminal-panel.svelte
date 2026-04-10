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
  import { createProjectConversationTerminalPanelState } from './project-conversation-terminal-panel-state.svelte'

  let {
    conversationId = '',
    workspacePath = '',
    selectedRepoPath = '',
    selectedFilePath = '',
  }: {
    conversationId?: string
    workspacePath?: string
    selectedRepoPath?: string
    selectedFilePath?: string
  } = $props()

  const terminal = createProjectConversationTerminalPanelState({
    getConversationId: () => conversationId,
    getWorkspacePath: () => workspacePath,
    getSelectedRepoPath: () => selectedRepoPath,
    getSelectedFilePath: () => selectedFilePath,
  })

  let terminalElement: HTMLDivElement | null = null

  onMount(() => {
    if (terminalElement) {
      void terminal.mount(terminalElement)
    }
    return () => {
      terminal.dispose()
    }
  })

  onDestroy(() => {
    terminal.dispose()
  })

  $effect(() => {
    terminal.syncConversation()
  })
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
        {terminal.lastLaunchLabel || terminal.contextTarget.label}
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        disabled={!conversationId || !terminal.terminalReady}
        onclick={() => void terminal.openTerminal('context')}
      >
        {#if terminal.status === 'connecting' && terminal.sessionID}
          <LoaderCircle class="mr-1.5 size-3.5 animate-spin" />
        {:else}
          <Terminal class="mr-1.5 size-3.5" />
        {/if}
        Open here
      </Button>
      <Button
        variant="ghost"
        size="sm"
        disabled={!conversationId || !terminal.terminalReady}
        onclick={() => void terminal.openTerminal('workspace-root')}
      >
        <RefreshCcw class="mr-1.5 size-3.5" />
        Workspace root
      </Button>
      <Button
        variant="ghost"
        size="sm"
        disabled={!terminal.sessionID}
        onclick={() => terminal.closeTerminal({ updateStatus: true })}
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
          terminal.status === 'error' && 'border-destructive/40',
        )}
      >
        <div class="flex items-start gap-2">
          {#if terminal.status === 'error'}
            <AlertCircle class="text-destructive mt-0.5 size-4 shrink-0" />
          {:else if terminal.status === 'connecting'}
            <LoaderCircle class="text-muted-foreground mt-0.5 size-4 shrink-0 animate-spin" />
          {:else}
            <SquareTerminal class="text-muted-foreground mt-0.5 size-4 shrink-0" />
          {/if}
          <div class="min-w-0">
            <p class="text-xs font-medium capitalize">{terminal.status}</p>
            <p class="text-muted-foreground text-xs">{terminal.statusMessage}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
