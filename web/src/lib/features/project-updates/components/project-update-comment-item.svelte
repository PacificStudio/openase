<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
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
    if (comment.isDeleted) {
      editing = false
    }
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
      if (success) {
        editing = false
      }
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    if (deleting || !window.confirm('Delete this comment?')) return

    deleting = true
    try {
      const success = (await onDelete?.(threadId, comment.id)) ?? false
      if (success) {
        editing = false
      }
    } finally {
      deleting = false
    }
  }
</script>

<div class="bg-muted/35 rounded-xl border px-4 py-3">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2 text-sm">
        <span class="font-medium">{comment.createdBy}</span>
        {#if comment.isDeleted}
          <Badge variant="outline" class="h-5 px-2 text-[10px]">Deleted</Badge>
        {/if}
      </div>
      <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-xs">
        <span>{formatRelativeTime(comment.createdAt)}</span>
        {#if isProjectUpdateEdited(comment.createdAt, comment.updatedAt, comment.editedAt)}
          <span
            >{projectUpdateEditedLabel(
              comment.createdAt,
              comment.updatedAt,
              comment.editedAt,
            )}</span
          >
        {/if}
      </div>
    </div>
    <div class="flex items-center gap-2">
      {#if !editing}
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Edit comment ${comment.id}`}
          onclick={beginEdit}
          disabled={comment.isDeleted || deleting}
        >
          <Pencil class="size-4" />
        </Button>
      {/if}
      <Button
        size="icon-sm"
        variant="ghost"
        aria-label={`Delete comment ${comment.id}`}
        onclick={handleDelete}
        disabled={comment.isDeleted || deleting}
      >
        <Trash2 class="size-4" />
      </Button>
    </div>
  </div>

  <div class="mt-3">
    {#if editing}
      <div class="space-y-3">
        <Textarea bind:value={editingBody} aria-label={`Edit comment ${comment.id}`} rows={4} />
        <div class="flex justify-end gap-2">
          <Button size="sm" variant="outline" onclick={cancelEdit} disabled={saving}>Cancel</Button>
          <Button size="sm" onclick={handleSave} disabled={!editingBody.trim() || saving}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>
    {:else if comment.isDeleted}
      <p class="text-muted-foreground text-sm italic">This comment was deleted.</p>
    {:else}
      <ProjectUpdateMarkdownContent source={comment.bodyMarkdown} />
    {/if}
  </div>
</div>
