<script lang="ts">
  import { MessageSquare } from '@lucide/svelte'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
  import { ProjectUpdateComposer, ProjectUpdateThreadCard } from '$lib/features/project-updates'
  import type { ProjectUpdateThread } from '$lib/features/project-updates'
  import type { ProjectUpdateStatus } from '$lib/features/project-updates'

  let {
    threads,
    loading,
    initialLoaded,
    creatingThread,
    loadError,
    hasMoreThreads,
    loadingMoreThreads,
    onSubmit,
    onLoadMoreThreads,
    onUpdateThread,
    onDeleteThread,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    threads: ProjectUpdateThread[]
    loading: boolean
    initialLoaded: boolean
    creatingThread: boolean
    loadError: string
    hasMoreThreads: boolean
    loadingMoreThreads: boolean
    onSubmit?: (draft: { status: ProjectUpdateStatus; body: string }) => Promise<boolean> | boolean
    onLoadMoreThreads?: () => Promise<boolean> | boolean
    onUpdateThread?: (
      threadId: string,
      draft: { status: ProjectUpdateStatus; body: string },
    ) => Promise<boolean> | boolean
    onDeleteThread?: (threadId: string) => Promise<boolean> | boolean
    onCreateComment?: (threadId: string, body: string) => Promise<boolean> | boolean
    onUpdateComment?: (
      threadId: string,
      commentId: string,
      body: string,
    ) => Promise<boolean> | boolean
    onDeleteComment?: (threadId: string, commentId: string) => Promise<boolean> | boolean
  } = $props()
</script>

<div class="flex min-h-0 flex-col">
  <div class="mb-2 flex items-center gap-1.5">
    <MessageSquare class="text-muted-foreground size-3.5" />
    <span class="text-foreground text-xs font-medium">Updates</span>
    {#if threads.length > 0}
      <span class="text-muted-foreground text-[11px]">{threads.length}</span>
    {/if}
  </div>

  <ProjectUpdateComposer creating={creatingThread} {onSubmit} />

  {#if loadError}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive mt-2 rounded-md border px-3 py-2 text-xs"
    >
      {loadError}
    </div>
  {/if}

  {#if loading && !initialLoaded}
    <div class="mt-2 space-y-2">
      {#each { length: 3 } as _}
        <div class="flex items-center gap-2">
          <Skeleton class="size-3.5 shrink-0 rounded-full" />
          <Skeleton class="h-3.5 w-3/4" />
        </div>
      {/each}
    </div>
  {:else if threads.length === 0}
    <div
      class="text-muted-foreground mt-2 flex flex-col items-center rounded-xl border border-dashed py-8 text-center text-xs"
    >
      <MessageSquare class="mb-2 size-4 opacity-40" />
      No updates yet
    </div>
  {:else}
    <div class="mt-2 space-y-1.5">
      {#each threads as thread (thread.id)}
        <ProjectUpdateThreadCard
          {thread}
          {onUpdateThread}
          {onDeleteThread}
          {onCreateComment}
          {onUpdateComment}
          {onDeleteComment}
        />
      {/each}

      {#if hasMoreThreads}
        <Button
          variant="ghost"
          size="sm"
          class="h-7 w-full text-xs"
          onclick={onLoadMoreThreads}
          disabled={loadingMoreThreads}
        >
          {loadingMoreThreads ? 'Loading…' : 'Load more'}
        </Button>
      {/if}
    </div>
  {/if}
</div>
