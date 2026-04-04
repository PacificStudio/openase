<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Textarea } from '$ui/textarea'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import { isProjectUpdateEdited, projectUpdateEditedLabel } from '../metadata'
  import type { ProjectUpdateComment } from '../types'
  import ProjectUpdateMarkdownContent from './project-update-markdown-content.svelte'

  let {
    threadId,
    comment,
    onUpdate,
    onDelete,
  }: {
    threadId: string
    comment: ProjectUpdateComment
    onUpdate?: (threadId: string, commentId: string, body: string) => Promise<boolean> | boolean
    onDelete?: (threadId: string, commentId: string) => Promise<boolean> | boolean
  } = $props()

  let editing = $state(false)
  let editingBody = $state('')
  let saving = $state(false)
  let deleting = $state(false)

  $effect(() => {
    editingBody = comment.bodyMarkdown
    if (comment.isDeleted) editing = false
  })

  function beginEdit() {
    editing = true
    editingBody = comment.bodyMarkdown
  }

  function cancelEdit() {
    editing = false
    editingBody = comment.bodyMarkdown
  }

  async function handleSave() {
    const body = editingBody.trim()
    if (!body || saving) return

    saving = true
    try {
      const success = (await onUpdate?.(threadId, comment.id, body)) ?? false
      if (success) editing = false
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (deleting || !window.confirm('Delete this comment?')) return

    deleting = true
    try {
      const success = (await onDelete?.(threadId, comment.id)) ?? false
      if (success) editing = false
    } finally {
      deleting = false
    }
  }
</script>

<div class={cn('group py-1.5', comment.isDeleted && 'opacity-50')}>
  {#if editing}
    <div class="space-y-2">
      <Textarea bind:value={editingBody} aria-label={`Edit comment ${comment.id}`} rows={3} />
      <div class="flex justify-end gap-1.5">
        <Button size="sm" variant="outline" onclick={cancelEdit} disabled={saving}>Cancel</Button>
        <Button size="sm" onclick={handleSave} disabled={!editingBody.trim() || saving}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
      </div>
    </div>
  {:else}
    <div class="flex items-start gap-2">
      <div class="min-w-0 flex-1">
        {#if comment.isDeleted}
          <p class="text-muted-foreground text-xs italic">This comment was deleted.</p>
        {:else}
          <ProjectUpdateMarkdownContent source={comment.bodyMarkdown} class="text-xs" />
        {/if}
        <div class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-1.5 text-[11px]">
          <span>{comment.createdBy}</span>
          <span class="opacity-40">&middot;</span>
          <span>{formatRelativeTime(comment.createdAt)}</span>
          {#if isProjectUpdateEdited(comment.createdAt, comment.updatedAt, comment.editedAt)}
            <span class="opacity-40">&middot;</span>
            <span>
              {projectUpdateEditedLabel(comment.createdAt, comment.updatedAt, comment.editedAt)}
            </span>
          {/if}
        </div>
      </div>
      {#if !comment.isDeleted}
        <div
          class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
        >
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded p-0.5 transition-colors"
            aria-label={`Edit comment ${comment.id}`}
            onclick={beginEdit}
            disabled={comment.isDeleted || deleting}
          >
            <Pencil class="size-3" />
          </button>
          <button
            type="button"
            class="text-muted-foreground hover:text-destructive rounded p-0.5 transition-colors"
            aria-label={`Delete comment ${comment.id}`}
            onclick={handleDelete}
            disabled={comment.isDeleted || deleting}
          >
            <Trash2 class="size-3" />
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
