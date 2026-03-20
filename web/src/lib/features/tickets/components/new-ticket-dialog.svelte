<script lang="ts">
  import { goto } from '$app/navigation'
  import type { TicketStatus, Workflow } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { createTicket, listStatuses, listWorkflows } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import {
    createNewTicketDraft,
    mapTicketStatusOptions,
    mapWorkflowOptions,
    parseNewTicketDraft,
    ticketPriorityOptions,
    type NewTicketDraft,
  } from '../new-ticket'

  let statuses = $state<TicketStatus[]>([])
  let workflows = $state<Workflow[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let error = $state('')
  let draft = $state<NewTicketDraft>(createNewTicketDraft([], []))
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
        error = ''
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
    error = ''

    try {
      const [statusPayload, workflowPayload] = await Promise.all([
        listStatuses(projectId),
        listWorkflows(projectId),
      ])
      if (requestId !== loadRequestId) return

      statuses = statusPayload.statuses
      workflows = workflowPayload.workflows
      draft = createNewTicketDraft(
        mapTicketStatusOptions(statusPayload.statuses),
        mapWorkflowOptions(workflowPayload.workflows),
      )
    } catch (caughtError) {
      if (requestId !== loadRequestId) return
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load ticket form options.'
    } finally {
      if (requestId === loadRequestId) {
        loading = false
      }
    }
  }

  function updateTextField(field: 'title' | 'description', event: Event) {
    const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
    draft = {
      ...draft,
      [field]: target.value,
    }
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()

    const projectId = appStore.currentProject?.id
    if (!projectId) {
      error = 'Select a project before creating a ticket.'
      return
    }

    const parsedDraft = parseNewTicketDraft(draft)
    if (!parsedDraft.ok) {
      error = parsedDraft.error
      return
    }

    saving = true
    error = ''

    try {
      const payload = await createTicket(projectId, parsedDraft.payload)
      const createdTicketId = payload.ticket.id

      appStore.closeNewTicketDialog()
      draft = createNewTicketDraft(statusOptions, workflowOptions)
      await goto('/tickets')
      appStore.openRightPanel({ type: 'ticket', id: createdTicketId })
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to create ticket.'
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
          oninput={(event) => updateTextField('title', event)}
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
          oninput={(event) => updateTextField('description', event)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-3">
        <div class="space-y-2">
          <Label>Status</Label>
          <Select.Root
            type="single"
            value={draft.statusId}
            disabled={loading || saving || statusOptions.length === 0}
            onValueChange={(value) => {
              draft = {
                ...draft,
                statusId: value || '',
              }
            }}
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
              draft = {
                ...draft,
                priority: value as NewTicketDraft['priority'],
              }
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
            onValueChange={(value) => {
              draft = {
                ...draft,
                workflowId: value || '',
              }
            }}
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

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || loading || !appStore.currentProject?.id}>
          {saving ? 'Creating…' : 'Create ticket'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
