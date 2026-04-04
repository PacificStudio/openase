<script lang="ts">
  import { cn } from '$lib/utils'
  import { Send } from '@lucide/svelte'
  import type { ProjectUpdateThread } from '../types'
  import ProjectUpdateCommentItem from './project-update-comment-item.svelte'

  let {
    thread,
    commentDraft = $bindable(''),
    creatingComment = false,
    showComments = $bindable(false),
    onCommentKeydown,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    thread: ProjectUpdateThread
    commentDraft?: string
    creatingComment?: boolean
    showComments?: boolean
    onCommentKeydown?: (event: KeyboardEvent) => void
    onCreateComment?: () => void
    onUpdateComment?: (
      threadId: string,
      commentId: string,
      body: string,
    ) => Promise<boolean> | boolean
    onDeleteComment?: (threadId: string, commentId: string) => Promise<boolean> | boolean
  } = $props()
</script>

{#if showComments || thread.commentCount === 0}
  {#if thread.comments.length > 0 || !thread.isDeleted}
    <div class="border-border/40 mt-1.5 space-y-0 border-t pt-1.5">
      {#each thread.comments as comment (comment.id)}
        <ProjectUpdateCommentItem
          threadId={thread.id}
          {comment}
          onUpdate={onUpdateComment}
          onDelete={onDeleteComment}
        />
      {/each}

      {#if !thread.isDeleted}
        <div class="flex items-center gap-2 pt-0.5">
          <input
            type="text"
            bind:value={commentDraft}
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
            onclick={onCreateComment}
            aria-label="Send reply"
          >
            <Send class="size-3" />
          </button>
        </div>
      {/if}
    </div>
  {/if}
{/if}
