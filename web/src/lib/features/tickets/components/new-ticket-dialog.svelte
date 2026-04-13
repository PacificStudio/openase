<script lang="ts">
  import { goto } from '$app/navigation'
  import type { TicketStatus } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { formatBoardPriorityLabel } from '$lib/features/board/public'
  import { createTicket, listProjectRepos, listStatuses } from '$lib/api/openase'
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import {
    createNewTicketDraft,
    mapProjectRepoOptions,
    mapTicketStatusOptions,
    parseNewTicketDraft,
    type NewTicketDraft,
  } from '../new-ticket'
  import NewTicketDialogMetadata from './new-ticket-dialog-metadata.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  const priorityLabels: Record<string, string> = {
    '': formatBoardPriorityLabel(''),
    urgent: formatBoardPriorityLabel('urgent'),
    high: formatBoardPriorityLabel('high'),
    medium: formatBoardPriorityLabel('medium'),
    low: formatBoardPriorityLabel('low'),
  }

  let statuses = $state<TicketStatus[]>([])
  let repoOptions = $state<ReturnType<typeof mapProjectRepoOptions>>([])
  let loading = $state(false)
  let saving = $state(false)
  let draft = $state<NewTicketDraft>(createNewTicketDraft([], []))
  let openProjectId = $state('')
  let loadRequestId = 0

  let statusPopoverOpen = $state(false)
  let priorityPopoverOpen = $state(false)
  let repoPopoverOpen = $state(false)
  let branchConfigOpen = $state(false)

  const statusOptions = $derived(mapTicketStatusOptions(statuses))

  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    const isOpen = appStore.newTicketDialogOpen

    if (!projectId || !isOpen) {
      openProjectId = ''
      if (!isOpen) {
        saving = false
        branchConfigOpen = false
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
          : i18nStore.t('tickets.newTicketDialog.errors.loadOptions'),
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
    const selected = draft.repoIds.includes(repoId)
    updateDraftField(
      'repoIds',
      selected ? draft.repoIds.filter((id) => id !== repoId) : [...draft.repoIds, repoId],
    )
    if (selected && draft.repoBranchOverrides[repoId] !== undefined) {
      const nextOverrides = { ...draft.repoBranchOverrides }
      delete nextOverrides[repoId]
      updateDraftField('repoBranchOverrides', nextOverrides)
    }
  }

  function updateRepoBranchOverride(repoId: string, value: string) {
    updateDraftField('repoBranchOverrides', {
      ...draft.repoBranchOverrides,
      [repoId]: value,
    })
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()

    const currentProject = appStore.currentProject
    if (!currentProject) {
      toastStore.error(i18nStore.t('tickets.newTicketDialog.errors.selectProject'))
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
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('tickets.newTicketDialog.errors.create'),
      )
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open={appStore.newTicketDialogOpen}>
  <Dialog.Content class="sm:max-w-xl">
    <Dialog.Header>
      <Dialog.Title>{i18nStore.t('tickets.newTicketDialog.title')}</Dialog.Title>
      <Dialog.Description>
        {#if appStore.currentProject}
          {i18nStore.t('tickets.newTicketDialog.description.withProject', {
            project: appStore.currentProject.name,
          })}
        {:else}
          {i18nStore.t('tickets.newTicketDialog.description.noProject')}
        {/if}
      </Dialog.Description>
    </Dialog.Header>

    <form class="flex min-h-0 flex-1 flex-col gap-6" onsubmit={handleSubmit}>
      <Dialog.Body class="space-y-4">
        <div class="space-y-2">
          <Label for="new-ticket-title">
            {i18nStore.t('tickets.newTicketDialog.labels.title')}
          </Label>
          <Input
            id="new-ticket-title"
            value={draft.title}
            placeholder={i18nStore.t('tickets.newTicketDialog.placeholders.title')}
            disabled={loading || saving}
            oninput={(event) =>
              updateDraftField('title', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-2">
          <Label for="new-ticket-description">
            {i18nStore.t('tickets.newTicketDialog.labels.description')}
          </Label>
          <Textarea
            id="new-ticket-description"
            value={draft.description}
            rows={4}
            placeholder={i18nStore.t('tickets.newTicketDialog.placeholders.description')}
            disabled={loading || saving}
            oninput={(event) =>
              updateDraftField('description', (event.currentTarget as HTMLTextAreaElement).value)}
          />
        </div>

        <NewTicketDialogMetadata
          {loading}
          {saving}
          {draft}
          {statusOptions}
          {repoOptions}
          {priorityLabels}
          bind:statusPopoverOpen
          bind:priorityPopoverOpen
          bind:repoPopoverOpen
          bind:branchConfigOpen
          onSelectStatus={selectStatus}
          onSelectPriority={selectPriority}
          onToggleRepoScope={toggleRepoScope}
          onUpdateRepoBranchOverride={updateRepoBranchOverride}
        />
      </Dialog.Body>

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || loading || !appStore.currentProject?.id}>
          {saving
            ? i18nStore.t('tickets.newTicketDialog.actions.creating')
            : i18nStore.t('tickets.newTicketDialog.actions.create')}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
