<script lang="ts">
  import { goto } from '$app/navigation'
  import type { TicketStatus } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createTicket, listProjectRepos, listStatuses } from '$lib/api/openase'
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import * as Dialog from '$ui/dialog'
  import * as Popover from '$ui/popover'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { ChevronDown, GitBranch } from '@lucide/svelte'
  import { PriorityIcon, StageIcon } from '$lib/features/board/public'
  import {
    createNewTicketDraft,
    mapProjectRepoOptions,
    mapTicketStatusOptions,
    parseNewTicketDraft,
    type TicketRepoOption,
    ticketPriorityOptions,
    type NewTicketDraft,
  } from '../new-ticket'

  const priorityLabels: Record<string, string> = {
    urgent: 'Urgent',
    high: 'High',
    medium: 'Medium',
    low: 'Low',
  }

  let statuses = $state<TicketStatus[]>([])
  let repoOptions = $state<TicketRepoOption[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let draft = $state<NewTicketDraft>(createNewTicketDraft([], []))
  let openProjectId = $state('')
  let loadRequestId = 0

  let statusPopoverOpen = $state(false)
  let priorityPopoverOpen = $state(false)
  let repoPopoverOpen = $state(false)

  const statusOptions = $derived(mapTicketStatusOptions(statuses))

  const selectedStatus = $derived(statusOptions.find((s) => s.id === draft.statusId) ?? null)
  const selectedRepoCount = $derived(draft.repoIds.length)

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    const isOpen = appStore.newTicketDialogOpen

    if (!projectId || !isOpen) {
      openProjectId = ''
      if (!isOpen) {
        saving = false
      }
      return
    }

    if (projectId === openProjectId) return
    openProjectId = projectId
    void loadDialogOptions(projectId)
  })

  async function loadDialogOptions(projectId: string) {
    const requestId = ++loadRequestId
    loading = true

    try {
      const [statusPayload, repoPayload] = await Promise.all([
        listStatuses(projectId),
        listProjectRepos(projectId),
      ])
      if (requestId !== loadRequestId) return

      const nextStatusOptions = mapTicketStatusOptions(statusPayload.statuses)
      const nextRepoOptions = mapProjectRepoOptions(repoPayload.repos)

      statuses = statusPayload.statuses
      repoOptions = nextRepoOptions
      draft = createNewTicketDraft(nextStatusOptions, nextRepoOptions)

      const defaultStatusId = appStore.newTicketDefaultStatusId
      if (defaultStatusId && nextStatusOptions.some((s) => s.id === defaultStatusId)) {
        draft = { ...draft, statusId: defaultStatusId }
      }
    } catch (caughtError) {
      if (requestId !== loadRequestId) return
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to load ticket form options.',
      )
    } finally {
      if (requestId === loadRequestId) {
        loading = false
      }
    }
  }

  function updateDraftField<K extends keyof NewTicketDraft>(field: K, value: NewTicketDraft[K]) {
    draft = { ...draft, [field]: value }
  }

  function selectStatus(statusId: string) {
    updateDraftField('statusId', statusId)
    statusPopoverOpen = false
  }

  function selectPriority(priority: NewTicketDraft['priority']) {
    updateDraftField('priority', priority)
    priorityPopoverOpen = false
  }

  function toggleRepoScope(repoId: string) {
    updateDraftField(
      'repoIds',
      draft.repoIds.includes(repoId)
        ? draft.repoIds.filter((id) => id !== repoId)
        : [...draft.repoIds, repoId],
    )
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()

    const currentProject = appStore.currentProject
    if (!currentProject) {
      toastStore.error('Select a project before creating a ticket.')
      return
    }
    const projectId = currentProject.id

    const parsedDraft = parseNewTicketDraft(draft, repoOptions)
    if (!parsedDraft.ok) {
      toastStore.error(parsedDraft.error)
      return
    }

    saving = true

    try {
      const payload = await createTicket(projectId, parsedDraft.payload)
      const createdTicketId = payload.ticket.id

      appStore.closeNewTicketDialog()
      draft = createNewTicketDraft(statusOptions, repoOptions)
      await goto(projectPath(currentProject.organization_id, currentProject.id, 'tickets'))
      appStore.openRightPanel({ type: 'ticket', id: createdTicketId })
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create ticket.',
      )
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open={appStore.newTicketDialogOpen}>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>Create Ticket</Dialog.Title>
      <Dialog.Description>
        {#if appStore.currentProject}
          Create a ticket in {appStore.currentProject.name}.
        {:else}
          Select a project before creating a ticket.
        {/if}
      </Dialog.Description>
    </Dialog.Header>

    <form class="space-y-4" onsubmit={handleSubmit}>
      <div class="space-y-2">
        <Label for="new-ticket-title">Title</Label>
        <Input
          id="new-ticket-title"
          value={draft.title}
          placeholder="Describe the outcome to deliver"
          disabled={loading || saving}
          oninput={(event) =>
            updateDraftField('title', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="new-ticket-description">Description</Label>
        <Textarea
          id="new-ticket-description"
          value={draft.description}
          rows={4}
          placeholder="Add implementation context, acceptance criteria, or constraints."
          disabled={loading || saving}
          oninput={(event) =>
            updateDraftField('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <!-- Metadata pickers row -->
      <div class="flex flex-wrap items-center gap-2">
        <!-- Status picker -->
        <Popover.Root bind:open={statusPopoverOpen}>
          <Popover.Trigger
            class={cn(
              'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
              (loading || saving) && 'pointer-events-none opacity-50',
            )}
            disabled={loading || saving || statusOptions.length === 0}
          >
            {#if selectedStatus}
              <StageIcon
                stage={selectedStatus.stage}
                color={selectedStatus.color}
                class="size-3.5"
              />
              <span class="text-foreground max-w-28 truncate">{selectedStatus.label}</span>
            {:else}
              <StageIcon stage="unstarted" class="size-3.5" />
              <span class="text-muted-foreground">Status</span>
            {/if}
            <ChevronDown class="text-muted-foreground size-3" />
          </Popover.Trigger>
          <Popover.Content align="start" class="w-48 gap-0 p-0.5">
            {#each statusOptions as option (option.id)}
              <button
                type="button"
                class={cn(
                  'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
                  option.id === draft.statusId && 'bg-muted',
                )}
                onclick={() => selectStatus(option.id)}
              >
                <StageIcon stage={option.stage} color={option.color} class="size-3.5" />
                <span class="text-foreground truncate">{option.label}</span>
              </button>
            {/each}
          </Popover.Content>
        </Popover.Root>

        <!-- Priority picker -->
        <Popover.Root bind:open={priorityPopoverOpen}>
          <Popover.Trigger
            class={cn(
              'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
              (loading || saving) && 'pointer-events-none opacity-50',
            )}
            disabled={loading || saving}
          >
            <PriorityIcon priority={draft.priority} class="size-3.5" />
            <span class="text-foreground">{priorityLabels[draft.priority]}</span>
            <ChevronDown class="text-muted-foreground size-3" />
          </Popover.Trigger>
          <Popover.Content align="start" class="w-36 gap-0 p-0.5">
            {#each ticketPriorityOptions as priority (priority)}
              <button
                type="button"
                class={cn(
                  'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
                  priority === draft.priority && 'bg-muted',
                )}
                onclick={() => selectPriority(priority)}
              >
                <PriorityIcon {priority} class="size-3.5" />
                <span class="text-foreground">{priorityLabels[priority]}</span>
              </button>
            {/each}
          </Popover.Content>
        </Popover.Root>

        <!-- Repo scope picker (multi-select) -->
        {#if repoOptions.length > 0}
          <Popover.Root bind:open={repoPopoverOpen}>
            <Popover.Trigger
              class={cn(
                'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
                (loading || saving) && 'pointer-events-none opacity-50',
              )}
              disabled={loading || saving}
            >
              <GitBranch class="text-muted-foreground size-3.5" />
              {#if selectedRepoCount === 0}
                <span class="text-muted-foreground">Repos</span>
              {:else if selectedRepoCount === 1}
                <span class="text-foreground max-w-28 truncate">
                  {repoOptions.find((r) => draft.repoIds.includes(r.id))?.label ?? '1 repo'}
                </span>
              {:else}
                <span class="text-foreground">{selectedRepoCount} repos</span>
              {/if}
              <ChevronDown class="text-muted-foreground size-3" />
            </Popover.Trigger>
            <Popover.Content align="start" class="max-h-56 w-64 gap-0 overflow-y-auto p-1">
              {#each repoOptions as option (option.id)}
                <label
                  class={cn(
                    'hover:bg-muted flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-1.5 text-xs transition-colors',
                  )}
                >
                  <Checkbox
                    class="size-3.5"
                    checked={draft.repoIds.includes(option.id)}
                    disabled={loading || saving}
                    onCheckedChange={() => toggleRepoScope(option.id)}
                  />
                  <div class="min-w-0 flex-1">
                    <span class="text-foreground truncate">{option.label}</span>
                    <span class="text-muted-foreground ml-1">({option.defaultBranch})</span>
                  </div>
                </label>
              {/each}
            </Popover.Content>
          </Popover.Root>
        {/if}
      </div>

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || loading || !appStore.currentProject?.id}>
          {saving ? 'Creating\u2026' : 'Create ticket'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
