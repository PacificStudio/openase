<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { listArchivedTickets, updateTicket } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import { Separator } from '$ui/separator'
  import { Archive, ArchiveRestore, Loader2 } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  type ArchivedTicket = {
    id: string
    identifier: string
    title: string
    statusName: string
    completedAt?: string
    createdAt: string
  }

  let tickets = $state<ArchivedTicket[]>([])
  let currentPage = $state(1)
  let totalTickets = $state(0)
  let loading = $state(false)
  let restoring = $state(false)
  let selectedIds = $state<Set<string>>(new Set())
  let loadedProjectId = ''
  let requestKey = $state('')

  const archivedTicketsPerPage = 20

  const allSelected = $derived(tickets.length > 0 && selectedIds.size === tickets.length)
  const someSelected = $derived(selectedIds.size > 0)
  const totalPages = $derived(Math.max(1, Math.ceil(totalTickets / archivedTicketsPerPage)))
  const pageStart = $derived(
    totalTickets === 0 ? 0 : (currentPage - 1) * archivedTicketsPerPage + 1,
  )
  const pageEnd = $derived(Math.min(totalTickets, currentPage * archivedTicketsPerPage))
  const canGoPrevious = $derived(currentPage > 1 && !loading && !restoring)
  const canGoNext = $derived(currentPage < totalPages && !loading && !restoring)

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    if (projectId === loadedProjectId) return
    loadedProjectId = projectId
    currentPage = 1
    totalTickets = 0
    tickets = []
    selectedIds = new Set()
    requestKey = ''
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    const nextRequestKey = `${projectId}:${currentPage}`
    if (nextRequestKey === requestKey) return
    requestKey = nextRequestKey
    void loadArchivedTickets(projectId, currentPage)
  })

  async function loadArchivedTickets(projectId: string, page: number) {
    loading = true
    selectedIds = new Set()

    try {
      const ticketPayload = await listArchivedTickets(projectId, {
        page,
        per_page: archivedTicketsPerPage,
      })

      totalTickets = ticketPayload.total
      if (ticketPayload.tickets.length === 0 && ticketPayload.total > 0 && page > 1) {
        currentPage = Math.max(1, Math.ceil(ticketPayload.total / archivedTicketsPerPage))
        requestKey = ''
        return
      }

      tickets = ticketPayload.tickets.map((t) => ({
        id: t.id,
        identifier: t.identifier,
        title: t.title,
        statusName: t.status_name,
        completedAt: t.completed_at ?? undefined,
        createdAt: t.created_at,
      }))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.archivedTickets.errors.load'),
      )
    } finally {
      loading = false
    }
  }

  function toggleSelect(ticketId: string) {
    const next = new Set(selectedIds)
    if (next.has(ticketId)) {
      next.delete(ticketId)
    } else {
      next.add(ticketId)
    }
    selectedIds = next
  }

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set()
    } else {
      selectedIds = new Set(tickets.map((t) => t.id))
    }
  }

  async function handleRestore() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const ids = [...selectedIds]
    if (ids.length === 0) return

    restoring = true

    try {
      await Promise.all(ids.map((id) => updateTicket(id, { archived: false })))
      toastStore.success(
        i18nStore.t('settings.archivedTickets.messages.restored', { count: ids.length }),
      )
      selectedIds = new Set()
      if (tickets.length === ids.length && currentPage > 1) {
        currentPage -= 1
      } else {
        requestKey = ''
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.archivedTickets.errors.restore'),
      )
    } finally {
      restoring = false
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">
      {i18nStore.t('settings.archivedTickets.heading')}
    </h2>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.archivedTickets.description')}
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground flex items-center gap-2 py-8 text-sm">
      <Loader2 class="size-4 animate-spin" />
      {i18nStore.t('settings.archivedTickets.status.loading')}
    </div>
  {:else if tickets.length === 0}
    <div class="flex flex-col items-center gap-2 py-12">
      <div class="bg-muted/60 flex size-10 items-center justify-center rounded-full">
        <Archive class="text-muted-foreground size-5" />
      </div>
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('settings.archivedTickets.messages.emptyTitle')}
      </p>
      <p class="text-muted-foreground text-sm">
        {i18nStore.t('settings.archivedTickets.messages.emptyDescription')}
      </p>
    </div>
  {:else}
    <div class="space-y-3">
      <!-- Toolbar -->
      <div class="flex items-center gap-3">
        <label class="flex cursor-pointer items-center gap-2">
          <Checkbox
            checked={allSelected}
            indeterminate={someSelected && !allSelected}
            onCheckedChange={toggleSelectAll}
          />
          <span class="text-muted-foreground text-xs">
            {#if someSelected}
              {i18nStore.t('settings.archivedTickets.toolbar.selected', {
                count: selectedIds.size,
              })}
            {:else}
              {i18nStore.t('settings.archivedTickets.toolbar.selectAll')}
            {/if}
          </span>
        </label>

        {#if someSelected}
          <Button
            size="sm"
            variant="outline"
            class="h-7 gap-1.5 text-xs"
            disabled={restoring}
            onclick={handleRestore}
          >
            <ArchiveRestore class="size-3" />
            {restoring
              ? i18nStore.t('settings.archivedTickets.toolbar.buttons.restoring')
              : i18nStore.t('settings.archivedTickets.toolbar.buttons.restore', {
                  count: selectedIds.size,
                })}
          </Button>
        {/if}
      </div>

      <div class="flex items-center justify-between gap-3">
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('settings.archivedTickets.pagination.summary', {
            start: pageStart,
            end: pageEnd,
            total: totalTickets,
          })}
        </p>
        <div class="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={!canGoPrevious}
            onclick={() => currentPage--}
          >
            {i18nStore.t('settings.archivedTickets.pagination.previous')}
          </Button>
          <span class="text-muted-foreground min-w-16 text-center text-xs">
            {i18nStore.t('settings.archivedTickets.pagination.pageCount', {
              current: currentPage,
              total: totalPages,
            })}
          </span>
          <Button size="sm" variant="outline" disabled={!canGoNext} onclick={() => currentPage++}>
            {i18nStore.t('settings.archivedTickets.pagination.next')}
          </Button>
        </div>
      </div>

      <!-- Ticket list -->
      <div class="border-border rounded-md border">
        {#each tickets as ticket, index (ticket.id)}
          {@const isSelected = selectedIds.has(ticket.id)}
          <button
            type="button"
            class={cn(
              'flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors',
              index > 0 && 'border-border border-t',
              isSelected ? 'bg-primary/5' : 'hover:bg-muted/50',
            )}
            onclick={() => toggleSelect(ticket.id)}
          >
            <Checkbox
              class="size-3.5 shrink-0"
              checked={isSelected}
              onCheckedChange={() => toggleSelect(ticket.id)}
            />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-muted-foreground shrink-0 font-mono text-[11px]">
                  {ticket.identifier}
                </span>
                <span class="text-foreground truncate text-xs font-medium">
                  {ticket.title}
                </span>
              </div>
              <div class="text-muted-foreground mt-0.5 flex items-center gap-2 text-[10px]">
                <span>{ticket.statusName}</span>
                <span>·</span>
                <span>{formatRelativeTime(ticket.completedAt ?? ticket.createdAt)}</span>
              </div>
            </div>
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>
