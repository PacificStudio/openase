<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { isProjectUpdateEdited, projectUpdateEditedLabel } from '../metadata'
  import type { ProjectUpdateThread } from '../types'
  import ProjectUpdateCommentItem from './project-update-comment-item.svelte'

  let {
    thread,
    showComments = false,
    commentDraft = '',
    creatingComment = false,
    onToggleComments,
    onCommentDraftChange,
    onCreateComment,
    onCommentKeydown,
    onUpdateComment,
    onDeleteComment,
  }: {
    thread: ProjectUpdateThread
    showComments?: boolean
    commentDraft?: string
    creatingComment?: boolean
    onToggleComments?: () => void
    onCommentDraftChange?: (value: string) => void
    onCreateComment?: () => void | Promise<void>
    onCommentKeydown?: (event: KeyboardEvent) => void
    onUpdateComment?: (
      threadId: string,
      commentId: string,
      body: string,
    ) => Promise<boolean> | boolean
    onDeleteComment?: (threadId: string, commentId: string) => Promise<boolean> | boolean
  } = $props()
</script>

<div class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-1.5 text-[11px]">
  <span>{thread.createdBy}</span>
  <span>&middot;</span>
  <span>{formatRelativeTime(thread.createdAt)}</span>
  {#if isProjectUpdateEdited(thread.createdAt, thread.updatedAt, thread.editedAt)}
    <span>&middot;</span>
    <span>{projectUpdateEditedLabel(thread.createdAt, thread.updatedAt, thread.editedAt)}</span>
  {/if}
  {#if thread.commentCount > 0}
    <span>&middot;</span>
    <button
      type="button"
      class="hover:text-foreground transition-colors"
      onclick={onToggleComments}
    >
      {thread.commentCount}
      {thread.commentCount === 1 ? 'reply' : 'replies'}
    </button>
  {/if}
</div>

{#if thread.isDeleted}
  <p class="text-muted-foreground mt-1.5 ml-6.5 text-xs italic">Deleted</p>
{/if}

{#if showComments || thread.commentCount === 0}
  {#if thread.comments.length > 0 || !thread.isDeleted}
    <div class="mt-2 ml-6.5 space-y-2">
      {#each thread.comments as comment (comment.id)}
        <ProjectUpdateCommentItem
          threadId={thread.id}
          {comment}
          onUpdate={onUpdateComment}
          onDelete={onDeleteComment}
        />
      {/each}

      {#if !thread.isDeleted}
        <div class="flex items-center gap-2">
          <input
            type="text"
            value={commentDraft}
            oninput={(event) =>
              onCommentDraftChange?.((event.currentTarget as HTMLInputElement).value)}
            onkeydown={onCommentKeydown}
            placeholder="Reply..."
            aria-label={`Reply to ${thread.title}`}
            class="text-foreground placeholder:text-muted-foreground min-w-0 flex-1 bg-transparent text-xs outline-none"
          />
          <button
            type="button"
            class={cn(
              'shrink-0 rounded p-1 transition-colors',
              commentDraft.trim() && !creatingComment
                ? 'text-primary hover:bg-primary/10'
                : 'text-muted-foreground/30 cursor-not-allowed',
            )}
            disabled={!commentDraft.trim() || creatingComment}
            onclick={() => void onCreateComment?.()}
            aria-label="Send reply"
          >
            <svg viewBox="0 0 24 24" class="size-3 fill-current" aria-hidden="true">
              <path d="M3.4 20.4 20.85 12 3.4 3.6l-.2 6.54 12.46 1.86-12.46 1.86.2 6.54Z" />
            </svg>
          </button>
        </div>
      {/if}
    </div>
  {/if}
{/if}
