<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { appStore } from '$lib/stores/app.svelte'
  import { listActivity, listTickets } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { subscribeProjectEvents } from '$lib/features/project-events'
  import {
    markProjectActivityCacheDirty,
    readProjectActivityCache,
    writeProjectActivityCache,
  } from '../activity-cache'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Activity, Search } from '@lucide/svelte'
  import type { ActivityEntry } from '../types'
  import { activityEventFilterOptions } from '../event-catalog'
  import {
    activityPageSize,
    mapActivityEntries,
    mergeActivityEntries,
    type ActivitySnapshotState,
  } from './activity-page-state'
  import ActivityPageLoading from './activity-page-loading.svelte'
  import ActivityLoadMorePlaceholder from './activity-load-more-placeholder.svelte'
  import ActivityTimeline from './activity-timeline.svelte'

  const loadMoreAnimationDurationMs = 240

  let entries = $state<ActivityEntry[]>([])
  let loading = $state(false)
  let loadingMore = $state(false)
  let error = $state('')
  let hasMore = $state(false)
  let nextCursor = $state('')
  let searchQuery = $state('')
  let selectedType = $state<string>('all')
  let initialLoaded = $state(false)
  let activeProjectId: string | null = null
  let requestVersion = 0
  let queuedReload = false
  let reloadInFlight = false

  const filtered = $derived(
    entries.filter((e) => {
      if (selectedType !== 'all' && e.eventType !== selectedType) return false
      if (searchQuery) {
        const q = searchQuery.toLowerCase()
        return (
          e.message.toLowerCase().includes(q) ||
          e.ticketIdentifier?.toLowerCase().includes(q) ||
          e.agentName?.toLowerCase().includes(q)
        )
      }
      return true
    }),
  )

  const isStaleLoad = (projectId: string, version: number) =>
    activeProjectId !== projectId || version !== requestVersion

  function writeSnapshot(projectId: string, snapshot: ActivitySnapshotState) {
    writeProjectActivityCache(projectId, snapshot)
  }

  function wait(durationMs: number) {
    return new Promise<void>((resolve) => {
      setTimeout(resolve, durationMs)
    })
  }

  async function loadActivityEntries(projectId: string, showLoading: boolean) {
    const version = ++requestVersion
    if (showLoading) loading = true
    error = ''

    try {
      const [activityPayload, ticketPayload] = await Promise.all([
        listActivity(projectId, { limit: activityPageSize }),
        listTickets(projectId),
      ])
      if (isStaleLoad(projectId, version)) {
        return
      }

      const nextEntries = mapActivityEntries(activityPayload, ticketPayload)
      const preservePagination = entries.length > 0
      const mergedEntries = preservePagination
        ? mergeActivityEntries(entries, nextEntries)
        : nextEntries
      const nextState = preservePagination ? nextCursor : activityPayload.next_cursor
      const nextHasMore = preservePagination ? hasMore : activityPayload.has_more

      entries = mergedEntries
      nextCursor = nextState
      hasMore = nextHasMore
      initialLoaded = true
      writeSnapshot(projectId, {
        entries: mergedEntries,
        nextCursor: nextState,
        hasMore: nextHasMore,
      })
    } catch (caughtError) {
      if (isStaleLoad(projectId, version)) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load activity.'
    } finally {
      if (!isStaleLoad(projectId, version)) {
        loading = false
      }
    }
  }

  async function handleLoadMore() {
    const projectId = activeProjectId
    if (!projectId || loadingMore || !hasMore || !nextCursor) {
      return
    }

    loadingMore = true
    error = ''
    const currentCursor = nextCursor

    try {
      const [[activityPayload, ticketPayload]] = await Promise.all([
        Promise.all([
          listActivity(projectId, { limit: activityPageSize, before: currentCursor }),
          listTickets(projectId),
        ]),
        wait(loadMoreAnimationDurationMs),
      ])
      if (activeProjectId !== projectId) {
        return
      }

      const olderEntries = mapActivityEntries(activityPayload, ticketPayload)
      const mergedEntries = mergeActivityEntries(entries, olderEntries)
      entries = mergedEntries
      nextCursor = activityPayload.next_cursor
      hasMore = activityPayload.has_more
      writeSnapshot(projectId, {
        entries: mergedEntries,
        nextCursor,
        hasMore,
      })
    } catch (caughtError) {
      if (activeProjectId !== projectId) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load more activity.'
    } finally {
      if (activeProjectId === projectId) {
        loadingMore = false
      }
    }
  }
  const requestReload = (projectId: string) => (
    (queuedReload = true),
    void drainReloadQueue(projectId)
  )

  async function drainReloadQueue(projectId: string) {
    if (!queuedReload || reloadInFlight || activeProjectId !== projectId) {
      return
    }

    reloadInFlight = true
    queuedReload = false
    try {
      await loadActivityEntries(projectId, false)
    } finally {
      reloadInFlight = false
      if (queuedReload && activeProjectId === projectId) {
        void drainReloadQueue(projectId)
      }
    }
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    activeProjectId = projectId ?? null
    queuedReload = false
    reloadInFlight = false
    if (!projectId) {
      entries = []
      initialLoaded = false
      loading = false
      loadingMore = false
      error = ''
      hasMore = false
      nextCursor = ''
      return
    }

    const cachedActivity = readProjectActivityCache(projectId)
    if (cachedActivity) {
      entries = cachedActivity.snapshot.entries
      nextCursor = cachedActivity.snapshot.nextCursor
      hasMore = cachedActivity.snapshot.hasMore
      initialLoaded = true
      loading = false
      loadingMore = false
      error = ''
      if (cachedActivity.dirty) void loadActivityEntries(projectId, false)
    } else {
      initialLoaded = false
      hasMore = false
      nextCursor = ''
      void loadActivityEntries(projectId, true)
    }

    const disconnect = subscribeProjectEvents(projectId, () => {
      markProjectActivityCacheDirty(projectId)
      requestReload(projectId)
    })

    return () => {
      if (activeProjectId === projectId) {
        activeProjectId = null
      }
      disconnect()
    }
  })
</script>

<PageScaffold title="Activity" description="Runtime events and agent lifecycle updates.">
  <div class="w-full space-y-4">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">
      <div class="relative min-w-0 flex-1">
        <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input placeholder="Search events..." class="pl-9" bind:value={searchQuery} />
      </div>
      <Select.Root
        type="single"
        onValueChange={(v) => {
          selectedType = v || 'all'
        }}
      >
        <Select.Trigger class="w-full sm:w-44">
          {activityEventFilterOptions.find((t) => t.value === selectedType)?.label ?? 'All events'}
        </Select.Trigger>
        <Select.Content class="max-h-72">
          {#each activityEventFilterOptions as t (t.value)}
            <Select.Item value={t.value}>{t.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    {#if loading && !initialLoaded}
      <ActivityPageLoading />
    {:else if error && entries.length === 0}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {:else if filtered.length > 0}
      <ActivityTimeline entries={filtered} />
    {:else}
      <div class="animate-fade-in-up flex flex-col items-center justify-center py-20">
        <div class="bg-muted/60 mb-4 flex size-12 items-center justify-center rounded-full">
          <Activity class="text-muted-foreground size-5" />
        </div>
        {#if entries.length === 0}
          <p class="text-foreground text-sm font-medium">No activity yet</p>
          <p class="text-muted-foreground mt-1 max-w-sm text-sm">
            Activity captures every agent run, ticket transition, and workflow event in real time.
            Create a ticket or trigger a workflow to see events here.
          </p>
        {:else}
          <p class="text-foreground text-sm font-medium">No matching events</p>
          <p class="text-muted-foreground mt-1 text-sm">
            Try adjusting your search or filter criteria.
          </p>
        {/if}
      </div>
    {/if}

    {#if initialLoaded && hasMore}
      <div class="space-y-3 pt-2">
        {#if loadingMore}
          <ActivityLoadMorePlaceholder />
        {/if}
        <div class="flex justify-center">
          <Button variant="outline" onclick={handleLoadMore} disabled={loadingMore}>
            {loadingMore ? 'Loading…' : 'Load more'}
          </Button>
        </div>
      </div>
    {/if}
  </div>
</PageScaffold>
