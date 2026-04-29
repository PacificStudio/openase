<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { appStore } from '$lib/stores/app.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
  import { MessageSquare } from '@lucide/svelte'
  import { createProjectUpdatesController } from '../project-updates-controller.svelte'
  import ProjectUpdateComposer from './project-update-composer.svelte'
  import ProjectUpdateThreadCard from './project-update-thread-card.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  const projectUpdates = createProjectUpdatesController({
    getProjectId: () => appStore.currentProject?.id ?? '',
  })
</script>

<PageScaffold
  title={i18nStore.t('projectUpdates.pageTitle')}
  description={i18nStore.t('projectUpdates.pageDescription')}
  helpSection="updates"
>
  <div class="w-full space-y-5">
    <div data-tour="updates-composer">
      <ProjectUpdateComposer
        creating={projectUpdates.creatingThread}
        onSubmit={projectUpdates.handleCreateThread}
      />
    </div>

    {#if projectUpdates.loadError}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {projectUpdates.loadError}
      </div>
    {/if}

    {#if projectUpdates.loading && !projectUpdates.initialLoaded}
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
    {:else if projectUpdates.threads.length === 0}
      <div
        class="flex flex-col items-center justify-center rounded-2xl border border-dashed py-18 text-center"
      >
        <div class="bg-muted/60 mb-4 flex size-12 items-center justify-center rounded-full">
          <MessageSquare class="text-muted-foreground size-5" />
        </div>
        <p class="text-sm font-medium">{i18nStore.t('projectUpdates.emptyState.title')}</p>
        <p class="text-muted-foreground mt-1 max-w-md text-sm">
          {i18nStore.t('projectUpdates.emptyState.description')}
        </p>
      </div>
    {:else}
      <div class="space-y-4" data-tour="project-updates-threads">
        {#each projectUpdates.threads as thread (thread.id)}
          <ProjectUpdateThreadCard
            {thread}
            onUpdateThread={projectUpdates.handleSaveThread}
            onDeleteThread={projectUpdates.handleDeleteThread}
            onCreateComment={projectUpdates.handleCreateComment}
            onUpdateComment={projectUpdates.handleSaveComment}
            onDeleteComment={projectUpdates.handleDeleteComment}
          />
        {/each}

        {#if projectUpdates.hasMoreThreads}
          <div class="flex justify-center">
            <Button
              variant="outline"
              class="min-w-36"
              onclick={() => void projectUpdates.handleLoadMoreThreads()}
              disabled={projectUpdates.loadingMoreThreads}
            >
              {projectUpdates.loadingMoreThreads
                ? i18nStore.t('projectUpdates.actions.loading')
                : i18nStore.t('projectUpdates.actions.loadMore')}
            </Button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</PageScaffold>
