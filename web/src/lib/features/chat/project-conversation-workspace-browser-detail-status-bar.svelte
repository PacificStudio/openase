<script lang="ts">
  import { ChevronLeft, ChevronRight, Timer, TimerOff, WrapText } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceFilePatch,
    ProjectConversationWorkspaceFilePreview,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import type { EditorWrapMode } from '$lib/components/code/wrap-mode'
  import * as Tooltip from '$ui/tooltip'
  import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state.svelte'

  let {
    activeEditorState,
    activePatch,
    activeFileLoading,
    activePreview,
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
    selectedChangedFilesCount: number
    showWrapToggle: boolean
    wrapMode: EditorWrapMode
    autosaveEnabled: boolean
    onToggleWrapMode: () => void
    onSelectPreviousChangedFile: () => void
    onSelectNextChangedFile: () => void
    onToggleAutosave: () => void
  } = $props()
</script>

<div
  class="border-border bg-muted/30 flex shrink-0 items-center gap-3 border-t px-3 py-1 text-[10px]"
  data-testid="workspace-browser-status-bar"
>
  {#if activeEditorState?.savePhase === 'saving'}
    <span class="font-medium text-sky-700 dark:text-sky-300">Saving…</span>
  {:else if activeEditorState?.savePhase === 'conflict'}
    <span class="font-medium text-amber-700 dark:text-amber-300">Conflict</span>
  {:else if activeEditorState?.externalChange}
    <span class="font-medium text-amber-700 dark:text-amber-300">Changed in workspace</span>
  {/if}
  {#if activePatch?.status && activePatch.status !== 'modified'}
    <span
      class={cn(
        'rounded px-1 font-bold uppercase',
        activePatch.status === 'added'
          ? 'text-emerald-600'
          : activePatch.status === 'deleted'
            ? 'text-rose-600'
            : 'text-sky-600',
      )}
    >
      {activePatch.status}
    </span>
  {/if}
  {#if selectedChangedFilesCount > 1}
    <span class="bg-border h-3 w-px"></span>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground"
      onclick={onSelectPreviousChangedFile}
      title="Previous changed file"
    >
      <ChevronLeft class="size-3" />
    </button>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground"
      onclick={onSelectNextChangedFile}
      title="Next changed file"
    >
      <ChevronRight class="size-3" />
    </button>
  {/if}

  <span class="flex-1"></span>

  {#if activeFileLoading}
    <span class="text-muted-foreground/50">Loading…</span>
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
            aria-label={wrapMode === 'wrap' ? 'Disable line wrap' : 'Enable line wrap'}
            aria-pressed={wrapMode === 'wrap'}
            data-testid="workspace-browser-wrap-toggle"
            onclick={onToggleWrapMode}
          >
            <WrapText class="size-3" />
          </button>
        {/snippet}
      </Tooltip.Trigger>
      <Tooltip.Content side="top" class="text-xs">
        {wrapMode === 'wrap'
          ? 'Word Wrap: On — click to disable'
          : 'Word Wrap: Off — click to enable'}
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
      {autosaveEnabled ? 'Autosave: On — click to disable' : 'Autosave: Off — click to enable'}
    </Tooltip.Content>
  </Tooltip.Root>
</div>
