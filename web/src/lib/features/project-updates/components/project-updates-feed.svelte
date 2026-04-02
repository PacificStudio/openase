<script lang="ts">
  import { cn } from '$lib/utils'
  import { Skeleton } from '$ui/skeleton'
  import { MessageSquare } from '@lucide/svelte'
  import type { ProjectUpdateStatus, ProjectUpdateThread } from '../types'
  import ProjectUpdateThreadCard from './project-update-thread-card.svelte'

  let {
    threads = [],
    loading = false,
    initialLoaded = false,
    error = '',
    notice = '',
    onUpdateThread,
    onDeleteThread,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
  }: {
    threads?: ProjectUpdateThread[]
    loading?: boolean
    initialLoaded?: boolean
    error?: string
    notice?: string
    onUpdateThread?: (
      threadId: string,
      draft: { status: ProjectUpdateStatus; title: string; body: string },
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

{#if error}
  <div
    class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
  >
    {error}
  </div>
{/if}

{#if notice}
  <div class="rounded-md border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
    {notice}
  </div>
{/if}

{#if loading && !initialLoaded}
  <div class="space-y-4">
    {#each { length: 2 } as _, i}
      <div class="border-border rounded-2xl border p-5 shadow-sm">
        <div class="space-y-3">
          <div class="flex items-center gap-2">
            <Skeleton class="h-5 w-20 rounded-full" />
            <Skeleton class="h-5 w-28 rounded-full" />
          </div>
          <Skeleton class={cn('h-6 rounded', i === 0 ? 'w-2/3' : 'w-1/2')} />
          <Skeleton class="h-4 w-48 rounded" />
          <div class="space-y-2">
            <Skeleton class="h-4 w-full rounded" />
            <Skeleton class="h-4 w-5/6 rounded" />
          </div>
        </div>
      </div>
    {/each}
  </div>
{:else if threads.length === 0}
  <div
    class="flex flex-col items-center justify-center rounded-2xl border border-dashed py-18 text-center"
  >
    <div class="bg-muted/60 mb-4 flex size-12 items-center justify-center rounded-full">
      <MessageSquare class="text-muted-foreground size-5" />
    </div>
    <p class="text-sm font-medium">No curated updates yet</p>
    <p class="text-muted-foreground mt-1 max-w-md text-sm">
      Post the first project update to capture current delivery status. Raw agent and workflow
      events continue to appear in Activity.
    </p>
  </div>
{:else}
  <div class="space-y-4">
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
  </div>
{/if}
