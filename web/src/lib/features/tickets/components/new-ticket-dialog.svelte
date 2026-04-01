<script lang="ts">
  import { goto } from '$app/navigation'
  import type { TicketStatus, Workflow } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createTicket, listProjectRepos, listStatuses, listWorkflows } from '$lib/api/openase'
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Checkbox } from '$ui/checkbox'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import {
    createNewTicketDraft,
    mapProjectRepoOptions,
    mapTicketStatusOptions,
    mapWorkflowOptions,
    parseNewTicketDraft,
    type TicketRepoOption,
    ticketPriorityOptions,
    type NewTicketDraft,
  } from '../new-ticket'

  let statuses = $state<TicketStatus[]>([])
  let workflows = $state<Workflow[]>([])
  let repoOptions = $state<TicketRepoOption[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let draft = $state<NewTicketDraft>(createNewTicketDraft([], [], []))
  let openProjectId = $state('')
  let loadRequestId = 0

  const statusOptions = $derived(mapTicketStatusOptions(statuses))
  const workflowOptions = $derived(mapWorkflowOptions(workflows))

  const selectedStatusLabel = $derived(
    statusOptions.find((option) => option.id === draft.statusId)?.label ?? 'Use project default',
  )
  const selectedWorkflowLabel = $derived(
    workflowOptions.find((option) => option.id === draft.workflowId)?.label ?? 'Unassigned',
  )

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
      const [statusPayload, workflowPayload, repoPayload] = await Promise.all([
        listStatuses(projectId),
        listWorkflows(projectId),
        listProjectRepos(projectId),
      ])
      if (requestId !== loadRequestId) return

      const nextStatusOptions = mapTicketStatusOptions(statusPayload.statuses)
      const nextWorkflowOptions = mapWorkflowOptions(workflowPayload.workflows)
      const nextRepoOptions = mapProjectRepoOptions(repoPayload.repos)

      statuses = statusPayload.statuses
      workflows = workflowPayload.workflows
      repoOptions = nextRepoOptions
      draft = createNewTicketDraft(nextStatusOptions, nextWorkflowOptions, nextRepoOptions)
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
    draft = {
      ...draft,
      [field]: value,
    }
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
      draft = createNewTicketDraft(statusOptions, workflowOptions, repoOptions)
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
          rows={6}
          placeholder="Add implementation context, acceptance criteria, or constraints."
          disabled={loading || saving}
          oninput={(event) =>
            updateDraftField('description', (event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-3">
        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.statusId}
            disabled={loading || saving || statusOptions.length === 0}
            onValueChange={(value) => updateDraftField('statusId', value || '')}
          >
            <Select.Trigger class="w-full">{selectedStatusLabel}</Select.Trigger>
            <Select.Content>
              {#each statusOptions as option (option.id)}
                <Select.Item value={option.id}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Priority</Label>
          <Select.Root
            type="single"
            value={draft.priority}
            disabled={loading || saving}
            onValueChange={(value) => {
              if (!value) return
              updateDraftField('priority', value as NewTicketDraft['priority'])
            }}
          >
            <Select.Trigger class="w-full capitalize">{draft.priority}</Select.Trigger>
            <Select.Content>
              {#each ticketPriorityOptions as priority (priority)}
                <Select.Item value={priority} class="capitalize">{priority}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Workflow</Label>
          <Select.Root
            type="single"
            value={draft.workflowId}
            disabled={loading || saving || workflowOptions.length === 0}
            onValueChange={(value) => updateDraftField('workflowId', value || '')}
          >
            <Select.Trigger class="w-full">{selectedWorkflowLabel}</Select.Trigger>
            <Select.Content>
              {#each workflowOptions as option (option.id)}
                <Select.Item value={option.id}>{option.label}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      {#if repoOptions.length > 1}
        <div class="space-y-3">
          <div class="space-y-1">
            <Label>Repository scopes</Label>
            <p class="text-muted-foreground text-xs">
              Multi-repo projects require an explicit repo scope selection before ticket creation.
            </p>
          </div>
          <div class="space-y-2 rounded-lg border p-3">
            {#each repoOptions as option (option.id)}
              <label
                class="hover:bg-muted/40 flex items-start gap-3 rounded-md px-2 py-2 transition-colors"
                for={`new-ticket-repo-${option.id}`}
              >
                <Checkbox
                  id={`new-ticket-repo-${option.id}`}
                  class="mt-0.5"
                  checked={draft.repoIds.includes(option.id)}
                  disabled={loading || saving}
                  onCheckedChange={() => toggleRepoScope(option.id)}
                />
                <div class="min-w-0 flex-1">
                  <div class="text-sm font-medium">{option.label}</div>
                  <div class="text-muted-foreground text-xs">
                    Default branch: {option.defaultBranch || 'main'}
                  </div>
                </div>
              </label>
            {/each}
          </div>
        </div>
      {:else if repoOptions.length === 1}
        <div class="rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-sm">
          <span class="text-muted-foreground">Repo scope:</span>
          <span class="text-foreground ml-1 font-medium">{repoOptions[0].label}</span>
        </div>
      {/if}

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || loading || !appStore.currentProject?.id}>
          {saving ? 'Creating…' : 'Create ticket'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
