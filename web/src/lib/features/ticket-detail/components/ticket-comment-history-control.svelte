<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import TicketMarkdownContent from './ticket-markdown-content.svelte'
  import type { TicketCommentRevision, TicketCommentTimelineItem } from '../types'
  import { formatRelativeTime } from '$lib/utils'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    comment,
    onLoad,
  }: {
    comment: TicketCommentTimelineItem
    onLoad?: (commentId: string) => Promise<TicketCommentRevision[]> | TicketCommentRevision[]
  } = $props()

  let open = $state(false)
  let revisions = $state<TicketCommentRevision[]>([])
  let loading = $state(false)
  let error = $state('')

  async function handleOpen() {
    open = true
    error = ''

    if (!onLoad) {
      error = i18nStore.t('ticketDetail.commentHistory.error.unavailable')
      return
    }

    if (revisions.length === comment.revisionCount) {
      return
    }

    loading = true

    try {
      revisions = (await onLoad(comment.commentId))
        .slice()
        .sort((left, right) => right.revisionNumber - left.revisionNumber)
    } catch (caughtError) {
      error =
        caughtError instanceof Error
          ? caughtError.message
          : i18nStore.t('ticketDetail.commentHistory.error.failed')
    } finally {
      loading = false
    }
  }
</script>

<Button
  size="xs"
  variant="ghost"
  aria-label={i18nStore.t('ticketDetail.commentHistory.actions.view', {
    actorName: comment.actor.name,
  })}
  onclick={() => void handleOpen()}
  disabled={loading}
>
  {loading
    ? i18nStore.t('ticketDetail.commentHistory.actions.loading')
    : i18nStore.t('ticketDetail.commentHistory.actions.history')}
</Button>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-2xl">
    <Dialog.Header>
      <Dialog.Title>{i18nStore.t('ticketDetail.commentHistory.dialog.title')}</Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('ticketDetail.commentHistory.dialog.description')}
      </Dialog.Description>
    </Dialog.Header>

    <div class="max-h-[70vh] space-y-3 overflow-y-auto pr-1">
      {#if loading}
        <p class="text-muted-foreground text-sm">
          {i18nStore.t('ticketDetail.commentHistory.status.loading')}
        </p>
      {:else if error}
        <p class="text-destructive text-sm">{error}</p>
      {:else if revisions.length === 0}
        <p class="text-muted-foreground text-sm">
          {i18nStore.t('ticketDetail.commentHistory.status.noHistory')}
        </p>
      {:else}
        {#each revisions as revision, index (revision.id)}
          <article class="border-border bg-background rounded-lg border">
            <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
              <div>
                <div class="text-sm font-medium">
                  {i18nStore.t('ticketDetail.commentHistory.revisionLabel', {
                    number: revision.revisionNumber,
                  })}
                </div>
                <div
                  class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[11px]"
                >
                  <span>{revision.editedBy}</span>
                  <span>{formatRelativeTime(revision.editedAt)}</span>
                  {#if index === 0}
                    <Badge variant="outline" class="h-5 px-2 text-[10px]">
                      {i18nStore.t('ticketDetail.commentHistory.badge.latest')}
                    </Badge>
                  {/if}
                </div>
              </div>
            </div>
            <div class="space-y-3 px-4 py-4">
              {#if revision.editReason}
                <p class="text-muted-foreground text-xs italic">{revision.editReason}</p>
              {/if}
              {#if revision.bodyMarkdown.trim()}
                <TicketMarkdownContent source={revision.bodyMarkdown} />
              {:else}
                <p class="text-muted-foreground text-sm italic">
                  {i18nStore.t('ticketDetail.commentHistory.status.noBody')}
                </p>
              {/if}
            </div>
          </article>
        {/each}
      {/if}
    </div>
  </Dialog.Content>
</Dialog.Root>
