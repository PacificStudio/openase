<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { createWorkflowWithBinding } from '../data'
  import { toggleWorkflowStatusSelection } from '../workflow-lifecycle'
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
  let error = $state('')
  let name = $state('')
  let agentId = $state('')
  let pickupStatusIds = $state<string[]>([])
  let finishStatusIds = $state<string[]>([])

  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === agentId)?.label ?? 'Select bound agent',
  )
  $effect(() => {
    if (!open) return

    name = `Workflow ${existingCount + 1}`
    agentId = agentOptions[0]?.id ?? ''
    pickupStatusIds = statuses[0] ? [statuses[0].id] : []
    finishStatusIds = statuses.at(-1) ? [statuses.at(-1)!.id] : statuses[0] ? [statuses[0].id] : []
    error = ''
  })

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    if (!projectId) {
      error = 'Select a project before creating a workflow.'
      return
    }
    if (!name.trim()) {
      error = 'Workflow name is required.'
      return
    }
    if (!agentId) {
      error = 'Bound agent is required.'
      return
    }
    if (pickupStatusIds.length === 0 || finishStatusIds.length === 0) {
      error = 'At least one pickup status and one finish status are required.'
      return
    }

    saving = true
    error = ''

    try {
      const payload = await createWorkflowWithBinding(
        projectId,
        {
          agentId,
          name: name.trim(),
          pickupStatusIds,
          finishStatusIds,
        },
        statuses,
        builtinRoleContent,
      )
      onCreated?.(payload)
      open = false
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to create workflow.'
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
          <Label>Pickup Statuses</Label>
          <div class="flex flex-wrap gap-2">
            {#each statuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  pickupStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving}
                onclick={() => (pickupStatusIds = toggleWorkflowStatusSelection(pickupStatusIds, status.id))}
              >
                {status.name}
              </button>
            {/each}
          </div>
        </div>

        <div class="space-y-2">
          <Label>Finish Statuses</Label>
          <div class="flex flex-wrap gap-2">
            {#each statuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  finishStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving}
                onclick={() => (finishStatusIds = toggleWorkflowStatusSelection(finishStatusIds, status.id))}
              >
                {status.name}
              </button>
            {/each}
          </div>
        </div>
      </div>

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || !projectId}>
          {saving ? 'Creating…' : 'Create workflow'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
