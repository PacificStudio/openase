<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { createWorkflowWithBinding } from '../data'
  import type { WorkflowAgentOption, WorkflowStatusOption, WorkflowSummary } from '../types'

  let {
    open = $bindable(false),
    projectId,
    statuses,
    agentOptions,
    existingCount,
    builtinRoleContent,
    onCreated,
  }: {
    open?: boolean
    projectId: string
    statuses: WorkflowStatusOption[]
    agentOptions: WorkflowAgentOption[]
    existingCount: number
    builtinRoleContent: string
    onCreated?: (payload: { workflow: WorkflowSummary; selectedId: string }) => void
  } = $props()

  let saving = $state(false)
  let name = $state('')
  let agentId = $state('')
  let pickupStatusId = $state('')
  let finishStatusId = $state('')

  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === agentId)?.label ?? 'Select bound agent',
  )
  const selectedPickupStatusLabel = $derived(
    statuses.find((status) => status.id === pickupStatusId)?.name ?? 'Select pickup status',
  )
  const selectedFinishStatusLabel = $derived(
    statuses.find((status) => status.id === finishStatusId)?.name ?? 'Select finish status',
  )

  $effect(() => {
    if (!open) return

    name = `Workflow ${existingCount + 1}`
    agentId = agentOptions[0]?.id ?? ''
    pickupStatusId = statuses[0]?.id ?? ''
    finishStatusId = statuses.at(-1)?.id ?? statuses[0]?.id ?? ''
  })

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    if (!projectId) {
      toastStore.error('Select a project before creating a workflow.')
      return
    }
    if (!name.trim()) {
      toastStore.error('Workflow name is required.')
      return
    }
    if (!agentId) {
      toastStore.error('Bound agent is required.')
      return
    }
    if (!pickupStatusId || !finishStatusId) {
      toastStore.error('Pickup and finish status are required.')
      return
    }

    saving = true

    try {
      const payload = await createWorkflowWithBinding(
        projectId,
        {
          agentId,
          name: name.trim(),
          pickupStatusId,
          finishStatusId,
        },
        statuses,
        builtinRoleContent,
      )
      onCreated?.(payload)
      open = false
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create workflow.',
      )
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>Create Workflow</Dialog.Title>
      <Dialog.Description>
        Bind a workflow to an explicit agent definition before it can dispatch.
      </Dialog.Description>
    </Dialog.Header>

    <form class="space-y-4" onsubmit={handleSubmit}>
      <div class="space-y-2">
        <Label for="workflow-create-name">Name</Label>
        <Input
          id="workflow-create-name"
          bind:value={name}
          disabled={saving}
          placeholder="Workflow name"
        />
      </div>

      <div class="space-y-2">
        <Label>Bound Agent</Label>
        <Select.Root
          type="single"
          value={agentId}
          disabled={saving || agentOptions.length === 0}
          onValueChange={(value) => (agentId = value || '')}
        >
          <Select.Trigger class="w-full">{selectedAgentLabel}</Select.Trigger>
          <Select.Content>
            {#each agentOptions as option (option.id)}
              <Select.Item value={option.id}>{option.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Pickup Status</Label>
          <Select.Root
            type="single"
            value={pickupStatusId}
            disabled={saving || statuses.length === 0}
            onValueChange={(value) => (pickupStatusId = value || '')}
          >
            <Select.Trigger class="w-full">{selectedPickupStatusLabel}</Select.Trigger>
            <Select.Content>
              {#each statuses as status (status.id)}
                <Select.Item value={status.id}>{status.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>

        <div class="space-y-2">
          <Label>Finish Status</Label>
          <Select.Root
            type="single"
            value={finishStatusId}
            disabled={saving || statuses.length === 0}
            onValueChange={(value) => (finishStatusId = value || '')}
          >
            <Select.Trigger class="w-full">{selectedFinishStatusLabel}</Select.Trigger>
            <Select.Content>
              {#each statuses as status (status.id)}
                <Select.Item value={status.id}>{status.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || !projectId}>
          {saving ? 'Creating…' : 'Create workflow'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
