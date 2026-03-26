<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import * as Select from '$ui/select'
  import {
    AlertCircle,
    CheckCircle2,
    Clock3,
    Layers3,
    Bot,
    Cpu,
    HardDrive,
    Power,
    RotateCcw,
    Trash2,
  } from '@lucide/svelte'
  import type { WorkflowAgentOption, WorkflowStatusOption, WorkflowSummary } from '../types'
  import {
    createWorkflowLifecycleDraft,
    parseWorkflowLifecycleDraft,
    type WorkflowLifecycleDraft,
    type WorkflowLifecyclePayload,
  } from '../workflow-lifecycle'
  import WorkflowNumberField from './workflow-number-field.svelte'

  const unchangedFinishStatusValue = '__unchanged__'

  let {
    workflow,
    statuses = [],
    agentOptions = [],
    saving = false,
    deleting = false,
    statusMessage = '',
    error = '',
    onSave,
    onDelete,
    class: className = '',
  }: {
    workflow: WorkflowSummary
    statuses?: WorkflowStatusOption[]
    agentOptions?: WorkflowAgentOption[]
    saving?: boolean
    deleting?: boolean
    statusMessage?: string
    error?: string
    onSave?: (payload: WorkflowLifecyclePayload) => void | Promise<void>
    onDelete?: () => void | Promise<void>
    class?: string
  } = $props()

  let draft = $state<WorkflowLifecycleDraft>({
    agentId: '',
    name: '',
    pickupStatusId: '',
    finishStatusId: '',
    maxConcurrent: '',
    maxRetryAttempts: '',
    timeoutMinutes: '',
    stallTimeoutMinutes: '',
    isActive: false,
  })
  let draftKey = $state('')
  let formError = $state('')

  const baseDraft = $derived(createWorkflowLifecycleDraft(workflow))
  const isDirty = $derived(
    draft.agentId !== baseDraft.agentId ||
      draft.name !== baseDraft.name ||
      draft.pickupStatusId !== baseDraft.pickupStatusId ||
      draft.finishStatusId !== baseDraft.finishStatusId ||
      draft.maxConcurrent !== baseDraft.maxConcurrent ||
      draft.maxRetryAttempts !== baseDraft.maxRetryAttempts ||
      draft.timeoutMinutes !== baseDraft.timeoutMinutes ||
      draft.stallTimeoutMinutes !== baseDraft.stallTimeoutMinutes ||
      draft.isActive !== baseDraft.isActive,
  )
  const selectedPickupStatusLabel = $derived(
    statuses.find((status) => status.id === draft.pickupStatusId)?.name ?? 'Select status',
  )
  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === draft.agentId)?.label ?? 'Select bound agent',
  )
  const selectedFinishStatusLabel = $derived(
    draft.finishStatusId
      ? (statuses.find((status) => status.id === draft.finishStatusId)?.name ?? 'Unknown status')
      : 'Leave unchanged',
  )
  const finishStatusValue = $derived(draft.finishStatusId || unchangedFinishStatusValue)
  const selectedAgent = $derived(agentOptions.find((option) => option.id === draft.agentId) ?? null)
  const machineSummary = $derived(
    workflow.requiredMachineLabels.length > 0
      ? `Labels: ${workflow.requiredMachineLabels.join(', ')}`
      : 'Local default machine',
  )

  $effect(() => {
    const nextKey = [
      workflow.id,
      workflow.version,
      workflow.agentId ?? '',
      workflow.name,
      workflow.isActive,
      workflow.pickupStatusId,
      workflow.finishStatusId ?? '',
      workflow.maxConcurrent,
      workflow.maxRetry,
      workflow.timeoutMinutes,
      workflow.stallTimeoutMinutes,
    ].join(':')

    if (nextKey === draftKey) return

    draft = createWorkflowLifecycleDraft(workflow)
    draftKey = nextKey
    formError = ''
  })

  function updateDraftField<K extends keyof WorkflowLifecycleDraft>(
    field: K,
    value: WorkflowLifecycleDraft[K],
  ) {
    draft = {
      ...draft,
      [field]: value,
    }
    formError = ''
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    formError = ''

    const parsed = parseWorkflowLifecycleDraft(draft)
    if (!parsed.ok) {
      formError = parsed.error
      return
    }

    await onSave?.(parsed.value)
  }

  async function handleDelete() {
    formError = ''
    if (!confirm(`Delete workflow "${workflow.name}"? This cannot be undone.`)) {
      return
    }

    await onDelete?.()
  }
</script>

<div class={cn('border-border flex h-full flex-col overflow-y-auto border-l', className)}>
  <div class="px-4 py-3">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="text-foreground text-sm font-medium">{workflow.name}</h3>
        <div class="text-muted-foreground mt-1 flex items-center gap-2 text-xs">
          <span class="capitalize">{workflow.type}</span>
          <span>v{workflow.version}</span>
          <span
            class={cn(
              'size-1.5 rounded-full',
              draft.isActive ? 'bg-emerald-500' : 'bg-neutral-500',
            )}
          ></span>
          <span>{draft.isActive ? 'Active' : 'Inactive'}</span>
        </div>
      </div>
      <Button
        type="button"
        variant={draft.isActive ? 'outline' : 'default'}
        size="sm"
        onclick={() => updateDraftField('isActive', !draft.isActive)}
      >
        <Power class="size-4" />
        {draft.isActive ? 'Deactivate' : 'Activate'}
      </Button>
    </div>
    <div class="text-muted-foreground mt-2 text-xs">
      Last modified {formatRelativeTime(workflow.lastModified)}
    </div>
  </div>

  <Separator />

  <form class="flex flex-1 flex-col" onsubmit={handleSubmit}>
    <div class="flex-1 space-y-4 px-4 py-4">
      <div class="space-y-2">
        <Label for="workflow-name">Name</Label>
        <Input
          id="workflow-name"
          value={draft.name}
          disabled={saving || deleting}
          oninput={(event) =>
            updateDraftField('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label>Bound Agent</Label>
        <Select.Root
          type="single"
          value={draft.agentId}
          disabled={saving || deleting || agentOptions.length === 0}
          onValueChange={(value) => updateDraftField('agentId', value || '')}
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
        <WorkflowNumberField
          id="workflow-max-concurrent"
          label="Max Concurrent"
          value={draft.maxConcurrent}
          icon={Layers3}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('maxConcurrent', value)}
        />
        <WorkflowNumberField
          id="workflow-max-retry"
          label="Max Retry Attempts"
          value={draft.maxRetryAttempts}
          icon={RotateCcw}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('maxRetryAttempts', value)}
        />
        <WorkflowNumberField
          id="workflow-timeout"
          label="Timeout Minutes"
          value={draft.timeoutMinutes}
          icon={Clock3}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('timeoutMinutes', value)}
        />
        <WorkflowNumberField
          id="workflow-stall-timeout"
          label="Stall Timeout Minutes"
          value={draft.stallTimeoutMinutes}
          icon={Clock3}
          disabled={saving || deleting}
          oninput={(value) => updateDraftField('stallTimeoutMinutes', value)}
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Provider</Label>
          <div
            class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
          >
            <Bot class="text-muted-foreground size-4" />
            <span>{selectedAgent?.providerName ?? 'Select an agent first'}</span>
          </div>
        </div>

        <div class="space-y-2">
          <Label>Model</Label>
          <div
            class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
          >
            <Cpu class="text-muted-foreground size-4" />
            <span>{selectedAgent?.modelName ?? 'Select an agent first'}</span>
          </div>
        </div>

        <div class="space-y-2">
          <Label>Machine</Label>
          <div
            class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
          >
            <HardDrive class="text-muted-foreground size-4" />
            <span>{machineSummary}</span>
          </div>
        </div>

        <div class="space-y-2">
          <Label>Workspace</Label>
          <div
            class="border-border text-foreground min-h-10 rounded-md border px-3 py-2 text-sm break-all"
          >
            {selectedAgent?.workspacePath || 'Platform-derived ticket workspace'}
          </div>
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-2">
          <Label>Pickup Status</Label>
          <Select.Root
            type="single"
            value={draft.pickupStatusId}
            disabled={saving || deleting || statuses.length === 0}
            onValueChange={(value) => updateDraftField('pickupStatusId', value || '')}
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
            value={finishStatusValue}
            disabled={saving || deleting}
            onValueChange={(value) =>
              updateDraftField(
                'finishStatusId',
                value === unchangedFinishStatusValue ? '' : (value ?? ''),
              )}
          >
            <Select.Trigger class="w-full">{selectedFinishStatusLabel}</Select.Trigger>
            <Select.Content>
              <Select.Item value={unchangedFinishStatusValue}>Leave unchanged</Select.Item>
              {#each statuses as status (status.id)}
                <Select.Item value={status.id}>{status.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      {#if statusMessage}
        <div class="flex items-center gap-2 text-xs text-emerald-400">
          <CheckCircle2 class="size-3.5" />
          {statusMessage}
        </div>
      {/if}

      {#if error || formError}
        <div class="text-destructive flex items-center gap-2 text-xs">
          <AlertCircle class="size-3.5" />
          {formError || error}
        </div>
      {/if}
    </div>

    <Separator />

    <div class="flex items-center justify-between gap-3 px-4 py-3">
      <Button
        type="button"
        variant="ghost"
        class="text-destructive hover:text-destructive"
        disabled={saving || deleting}
        onclick={() => void handleDelete()}
      >
        <Trash2 class="size-4" />
        {deleting ? 'Deleting…' : 'Delete Workflow'}
      </Button>

      <Button type="submit" size="sm" disabled={!isDirty || saving || deleting}>
        {saving ? 'Saving…' : 'Save Changes'}
      </Button>
    </div>
  </form>
</div>
