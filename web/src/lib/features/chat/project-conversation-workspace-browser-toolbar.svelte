<script lang="ts">
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import { Check, Copy, FolderTree, RefreshCcw, SquareTerminal, X } from '@lucide/svelte'
  import { chatT } from './i18n'

  let {
    workspacePath = '',
    pathCopied = false,
    showTerminalButton = false,
    terminalPanelOpen = false,
    conversationId = '',
    metadataLoading = false,
    onCopyWorkspacePath,
    onToggleTerminal,
    onRefreshWorkspace,
    onClose,
  }: {
    workspacePath?: string
    pathCopied?: boolean
    showTerminalButton?: boolean
    terminalPanelOpen?: boolean
    conversationId?: string
    metadataLoading?: boolean
    onCopyWorkspacePath?: () => void
    onToggleTerminal?: () => void
    onRefreshWorkspace?: () => void
    onClose?: () => void
  } = $props()
</script>

<div class="border-border flex h-11 items-center gap-1.5 border-b px-3">
  <FolderTree class="text-muted-foreground size-3 shrink-0" />
  <span class="text-[12px] font-semibold">{chatT('chat.workspaceLabel')}</span>
  {#if workspacePath}
    <button
      type="button"
      class="text-muted-foreground/50 hover:text-muted-foreground group flex min-w-0 items-center gap-1 truncate text-[11px] transition-colors"
      title={chatT('chat.copyWorkspacePath')}
      onclick={onCopyWorkspacePath}
    >
      <span class="min-w-0 truncate">{workspacePath}</span>
      {#if pathCopied}
        <Check class="size-2.5 shrink-0 text-emerald-500" />
      {:else}
        <Copy class="size-2.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
      {/if}
    </button>
  {/if}

  <div class="flex-1"></div>
  {#if showTerminalButton}
    <Button
      variant={terminalPanelOpen ? 'secondary' : 'ghost'}
      size="icon-xs"
      class={cn('text-muted-foreground size-6', terminalPanelOpen && 'text-foreground')}
      aria-label={chatT('chat.toggleTerminal')}
      onclick={onToggleTerminal}
      disabled={!conversationId}
    >
      <SquareTerminal class="size-3" />
    </Button>
  {/if}
  <Button
    variant="ghost"
    size="icon-xs"
    class="text-muted-foreground size-6"
    aria-label={chatT('chat.refreshWorkspaceBrowser')}
    onclick={onRefreshWorkspace}
    disabled={!conversationId || metadataLoading}
  >
    <RefreshCcw class={cn('size-3', metadataLoading && 'animate-spin')} />
  </Button>
  {#if onClose}
    <Button
      variant="ghost"
      size="icon-xs"
      class="text-muted-foreground size-6"
      aria-label={chatT('chat.closeWorkspaceBrowser')}
      onclick={onClose}
    >
      <X class="size-3" />
    </Button>
  {/if}
</div>
