<script lang="ts">
  import type { ProjectConversationWorkspaceDiffFile } from '$lib/api/chat'
  import { cn } from '$lib/utils'
import { chatT } from './i18n'
  import { ChevronRight, Minus, Plus, Undo2 } from '@lucide/svelte'
  import { fileIcon, formatTotals } from './project-conversation-workspace-browser-helpers'
  import {
    dirtyFileColorClass,
    filenameFromPath,
  } from './project-conversation-workspace-browser-sidebar-helpers'

  let {
    title,
    count,
    files,
    selectedFilePath = '',
    mode,
    actionPendingPath = '',
    commitPending = false,
    headerActionTitle,
    headerActionTestId,
    onHeaderAction,
    onSelectFile,
    onStage,
    onUnstage,
    onDiscard,
  }: {
    title: string
    count: number
    files: ProjectConversationWorkspaceDiffFile[]
    selectedFilePath?: string
    mode: 'staged' | 'unstaged'
    actionPendingPath?: string
    commitPending?: boolean
    headerActionTitle: string
    headerActionTestId: string
    onHeaderAction?: () => Promise<void>
    onSelectFile?: (path: string) => void
    onStage?: (path: string) => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onDiscard?: (path: string) => Promise<void>
  } = $props()

  let sectionOpen = $state(true)
  const disabled = $derived(actionPendingPath !== '' || commitPending)

  function statusBadge(file: ProjectConversationWorkspaceDiffFile) {
    if (file.status === 'deleted') return { label: 'D', className: 'text-red-500' }
    if (file.status === 'added') return { label: 'A', className: 'text-emerald-600' }
    if (file.status === 'untracked') return { label: 'U', className: 'text-emerald-600' }
    if (file.status === 'renamed') return { label: 'R', className: 'text-sky-500' }
    return { label: 'M', className: 'text-amber-500' }
  }
</script>

<div class="border-border border-t">
  <div class="flex items-center gap-0.5 pr-1">
    <button
      type="button"
      class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-0.5 text-[10px] font-semibold tracking-wider uppercase transition-colors"
      onclick={() => (sectionOpen = !sectionOpen)}
    >
      <ChevronRight
        class={cn('size-2.5 shrink-0 transition-transform duration-100', sectionOpen && 'rotate-90')}
      />
      {title}
      {#if mode === 'staged'}
        <span class="sr-only">{count} staged</span>
      {/if}
      <span
        class="text-muted-foreground/50 ml-auto text-[9px] font-normal tracking-normal normal-case"
      >
        {count}
      </span>
    </button>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 transition-colors"
      title={headerActionTitle}
      data-testid={headerActionTestId}
      disabled={disabled}
      onclick={() => void onHeaderAction?.()}
    >
      {#if mode === 'staged'}<Minus class="size-3"></Minus>{:else}<Plus class="size-3"></Plus>{/if}
    </button>
  </div>

  {#if sectionOpen}
    {#each files as file (file.path)}
      {@const badge = statusBadge(file)}
      <div
        class={cn(
          'hover:bg-muted/40 group flex items-center gap-1 py-[3px] pr-2 pl-5 text-[11px] transition-colors',
          file.path === selectedFilePath && 'bg-primary/10',
        )}
      >
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-1.5 text-left"
          title={chatT('chat.workspace.changeSummary', { path: file.path, status: file.status, added: file.added, removed: file.removed })}
          data-testid={`workspace-branch-file-${file.path}`}
          onclick={() => onSelectFile?.(file.path)}
        >
          {#each [fileIcon(filenameFromPath(file.path))] as info}
            <info.icon class={cn('size-3 shrink-0', info.colorClass)} />
          {/each}
          <span class={cn('min-w-0 truncate', dirtyFileColorClass(file.status))}>
            {filenameFromPath(file.path)}
          </span>
          {#if file.path.includes('/')}
            <span class="text-muted-foreground/30 min-w-0 shrink truncate text-[9px]">
              {file.path.slice(0, file.path.lastIndexOf('/'))}
            </span>
          {/if}
        </button>
        <span class="text-muted-foreground/40 shrink-0 text-[9px]">
          {formatTotals(file.added, file.removed)}
        </span>
        <span class={cn('shrink-0 font-mono text-[9px] font-bold', badge.className)}>
          {badge.label}
        </span>

        {#if mode === 'staged'}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 opacity-0 transition-all group-hover:opacity-100"
            title={chatT('chat.workspace.actions.unstage')}
            data-testid={`workspace-branch-unstage-${file.path}`}
            disabled={disabled}
            onclick={(event) => {
              event.stopPropagation()
              void onUnstage?.(file.path)
            }}
          >
            <Minus class="size-3" />
          </button>
        {:else}
          <div class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground hover:bg-muted rounded p-0.5 transition-colors"
              title={chatT('chat.workspace.actions.stage')}
              data-testid={`workspace-branch-stage-${file.path}`}
              disabled={disabled}
              onclick={(event) => {
                event.stopPropagation()
                void onStage?.(file.path)
              }}
            >
              <Plus class="size-3" />
            </button>
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground hover:bg-muted rounded p-0.5 transition-colors"
              title={chatT('chat.workspace.actions.discard')}
              data-testid={`workspace-branch-discard-${file.path}`}
              disabled={disabled}
              onclick={(event) => {
                event.stopPropagation()
                void onDiscard?.(file.path)
              }}
            >
              <Undo2 class="size-3" />
            </button>
          </div>
        {/if}
      </div>
    {/each}
  {/if}
</div>
