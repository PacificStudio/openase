<script lang="ts">
  import { ChevronLeft, ChevronRight, Timer, TimerOff, WrapText } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceDiffFile,
    ProjectConversationWorkspaceFilePatch,
    ProjectConversationWorkspaceFilePreview,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import type { EditorWrapMode } from '$lib/components/code/wrap-mode'
  import * as Tooltip from '$ui/tooltip'
  import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state.svelte'
  import { chatT } from './i18n'

  let {
    activeEditorState,
    activePatch,
    activeFileLoading,
    activePreview,
    activeChangedFile,
    selectedChangedFilesCount,
    showWrapToggle,
    wrapMode,
    autosaveEnabled,
    onToggleWrapMode,
    onSelectPreviousChangedFile,
    onSelectNextChangedFile,
    onToggleAutosave,
  }: {
    activeEditorState: WorkspaceFileEditorState | null
    activePatch: ProjectConversationWorkspaceFilePatch | null
    activeFileLoading: boolean
    activePreview: ProjectConversationWorkspaceFilePreview | null
    activeChangedFile: ProjectConversationWorkspaceDiffFile | null
    selectedChangedFilesCount: number
    showWrapToggle: boolean
    wrapMode: EditorWrapMode
    autosaveEnabled: boolean
    onToggleWrapMode: () => void
    onSelectPreviousChangedFile: () => void
    onSelectNextChangedFile: () => void
    onToggleAutosave: () => void
  } = $props()

  const statusLabel = $derived(activeChangedFile?.status ?? activePatch?.status ?? '')
  const statusBadge = $derived.by(() => {
    switch (statusLabel) {
      case 'added':
        return 'A'
      case 'deleted':
        return 'D'
      case 'renamed':
        return 'R'
      case 'untracked':
        return 'U'
      case 'modified':
        return 'M'
      default:
        return ''
    }
  })
  const statusTone = $derived.by(() => {
    switch (statusLabel) {
      case 'added':
      case 'untracked':
        return 'text-emerald-700 dark:text-emerald-300'
      case 'deleted':
        return 'text-rose-700 dark:text-rose-300'
      case 'renamed':
        return 'text-sky-700 dark:text-sky-300'
      case 'modified':
        return 'text-amber-700 dark:text-amber-300'
      default:
        return 'text-muted-foreground/60'
    }
  })
</script>

<div
  class="border-border bg-muted/30 flex shrink-0 items-center gap-3 border-t px-3 py-1 text-[10px]"
  data-testid="workspace-browser-status-bar"
>
  {#if activeEditorState?.savePhase === 'saving'}
    <span class="font-medium text-sky-700 dark:text-sky-300">{chatT('chat.saving')}</span>
  {:else if activeEditorState?.savePhase === 'conflict'}
    <span class="font-medium text-amber-700 dark:text-amber-300">
      {chatT('chat.workspaceConflict')}
    </span>
  {:else if activeEditorState?.externalChange}
    <span class="font-medium text-amber-700 dark:text-amber-300">
      {chatT('chat.workspaceChanged')}
    </span>
  {/if}
  {#if selectedChangedFilesCount > 1}
    <span class="bg-border h-3 w-px"></span>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground"
      onclick={onSelectPreviousChangedFile}
      title={chatT('chat.previousChangedFile')}
    >
      <ChevronLeft class="size-3" />
    </button>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground"
      onclick={onSelectNextChangedFile}
      title={chatT('chat.nextChangedFile')}
    >
      <ChevronRight class="size-3" />
    </button>
  {/if}
  {#if statusBadge}
    <span class="bg-border h-3 w-px"></span>
    <span
      class={cn('font-mono font-semibold', statusTone)}
      data-testid="workspace-browser-status-badge"
    >
      {statusBadge}
    </span>
    <span class={statusTone} data-testid="workspace-browser-status-label">{statusLabel}</span>
    {#if activeChangedFile}
      <span class="text-muted-foreground/60" data-testid="workspace-browser-status-totals">
        +{activeChangedFile.added} -{activeChangedFile.removed}
      </span>
    {/if}
  {/if}

  <span class="flex-1"></span>

  {#if activeFileLoading}
    <span class="text-muted-foreground/50">{chatT('chat.loadingEllipsis')}</span>
  {/if}
  {#if activePreview}
    <span class="text-muted-foreground/60">{activePreview.mediaType}</span>
    <span class="text-muted-foreground/60">{activePreview.sizeBytes} B</span>
  {/if}
  {#if showWrapToggle}
    <span class="bg-border h-3 w-px"></span>
    <Tooltip.Root>
      <Tooltip.Trigger>
        {#snippet child({ props })}
          <button
            {...props}
            type="button"
            class={cn(
              'inline-flex items-center gap-0.5 transition-colors',
              wrapMode === 'wrap'
                ? 'text-foreground'
                : 'text-muted-foreground/60 hover:text-foreground',
            )}
            aria-label={wrapMode === 'wrap'
              ? chatT('chat.disableLineWrap')
              : chatT('chat.enableLineWrap')}
            aria-pressed={wrapMode === 'wrap'}
            data-testid="workspace-browser-wrap-toggle"
            onclick={onToggleWrapMode}
          >
            <WrapText class="size-3" />
          </button>
        {/snippet}
      </Tooltip.Trigger>
      <Tooltip.Content side="top" class="text-xs">
        {wrapMode === 'wrap' ? chatT('chat.wordWrapOnHint') : chatT('chat.wordWrapOffHint')}
      </Tooltip.Content>
    </Tooltip.Root>
  {/if}
  <span class="bg-border h-3 w-px"></span>
  <Tooltip.Root>
    <Tooltip.Trigger>
      {#snippet child({ props })}
        <button
          {...props}
          type="button"
          class={cn(
            'inline-flex items-center transition-colors',
            autosaveEnabled ? 'text-foreground' : 'text-muted-foreground/60 hover:text-foreground',
          )}
          onclick={onToggleAutosave}
          data-testid="workspace-browser-autosave-toggle"
        >
          {#if autosaveEnabled}
            <Timer class="size-3" />
          {:else}
            <TimerOff class="size-3" />
          {/if}
        </button>
      {/snippet}
    </Tooltip.Trigger>
    <Tooltip.Content side="top" class="text-xs">
      {autosaveEnabled ? chatT('chat.autosaveOnHint') : chatT('chat.autosaveOffHint')}
    </Tooltip.Content>
  </Tooltip.Root>
</div>
