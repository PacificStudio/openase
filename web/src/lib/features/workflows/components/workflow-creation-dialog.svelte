<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as Collapsible from '$ui/collapsible'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { ChevronRight } from '@lucide/svelte'
  import { createWorkflowWithBinding } from '../data'
  import { parseHarnessTemplateStatusBindings } from '../model'
  import {
    createWorkflowHooksDraft,
    parseWorkflowHooksDraft,
    validateWorkflowHooksDraft,
    type WorkflowHooksDraft,
  } from '../workflow-hooks'
  import { toggleWorkflowStatusSelection } from '../workflow-lifecycle'
  import type {
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
    WorkflowTemplateDraft,
  } from '../types'
  import WorkflowHooksEditor from './workflow-hooks-editor.svelte'

  let {
    open = $bindable(false),
    projectId,
    statuses,
    agentOptions,
    existingCount,
    builtinRoleContent,
    templateDraft = null,
    onCreated,
  }: {
    open?: boolean
    projectId: string
    statuses: WorkflowStatusOption[]
    agentOptions: WorkflowAgentOption[]
    existingCount: number
    builtinRoleContent: string
    templateDraft?: WorkflowTemplateDraft | null
    onCreated?: (payload: { workflow: WorkflowSummary; selectedId: string }) => void
  } = $props()

  let saving = $state(false)
  let name = $state('')
  let agentId = $state('')
  let pickupStatusIds = $state<string[]>([])
  let finishStatusIds = $state<string[]>([])
  let templateStatusError = $state('')
  let advancedOpen = $state(false)
  let hookDraft = $state<WorkflowHooksDraft>(createWorkflowHooksDraft())
  let hookError = $state('')
  let wasOpen = false

  const selectedAgentLabel = $derived(
    agentOptions.find((option) => option.id === agentId)?.label ?? 'Select bound agent',
  )
  const selectableStatuses = $derived(statuses)
  const hookValidation = $derived(validateWorkflowHooksDraft(hookDraft))
  $effect(() => {
    if (open && !wasOpen) {
      name = templateDraft?.name ?? `Workflow ${existingCount + 1}`
      agentId = agentOptions[0]?.id ?? ''
      templateStatusError = ''
      advancedOpen = false
      hookDraft = createWorkflowHooksDraft()
      hookError = ''

      const defaultStatusIds = selectableStatuses[0] ? [selectableStatuses[0].id] : []
      pickupStatusIds = defaultStatusIds
      finishStatusIds = defaultStatusIds

      if (templateDraft) {
        try {
          const bindings = parseHarnessTemplateStatusBindings(templateDraft.content)
          const pickupResolution = resolveTemplateStatusIds(bindings.pickupStatusNames)
          const finishResolution = resolveTemplateStatusIds(bindings.finishStatusNames)
          const missingNames = [
            ...new Set([...pickupResolution.missingNames, ...finishResolution.missingNames]),
          ]

          if (missingNames.length > 0) {
            templateStatusError = `Template status bindings are not configured in this project: ${missingNames.join(', ')}.`
            pickupStatusIds = []
            finishStatusIds = []
          } else {
            if (pickupResolution.ids.length > 0) pickupStatusIds = pickupResolution.ids
            if (finishResolution.ids.length > 0) finishStatusIds = finishResolution.ids
          }
        } catch (caughtError) {
          templateStatusError =
            caughtError instanceof Error
              ? caughtError.message
              : 'Failed to parse workflow template status bindings.'
          pickupStatusIds = []
          finishStatusIds = []
        }
      }
    }

    wasOpen = open
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
    if (templateStatusError) {
      toastStore.error(templateStatusError)
      return
    }
    if (pickupStatusIds.length === 0 || finishStatusIds.length === 0) {
      toastStore.error('Pickup and finish status are required.')
      return
    }

    const parsedHooks = parseWorkflowHooksDraft(hookDraft)
    if (!parsedHooks.ok) {
      hookError = parsedHooks.error
      advancedOpen = true
      return
    }

    saving = true

    try {
      const payload = await createWorkflowWithBinding(
        projectId,
        {
          agentId,
          name: name.trim(),
          workflowType: templateDraft?.workflowType ?? 'coding',
          harnessPath: templateDraft?.harnessPath ?? null,
          pickupStatusIds,
          finishStatusIds,
          hooks: parsedHooks.value,
        },
        statuses,
        templateDraft?.content ?? builtinRoleContent,
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

  function resolveTemplateStatusIds(names: string[]) {
    const ids: string[] = []
    const missingNames: string[] = []

    for (const name of names) {
      const status = selectableStatuses.find(
        (item) => item.name.trim().toLowerCase() === name.trim().toLowerCase(),
      )
      if (!status) {
        missingNames.push(name)
        continue
      }
      if (!ids.includes(status.id)) {
        ids.push(status.id)
      }
    }

    return { ids, missingNames }
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
            {#each selectableStatuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  pickupStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving}
                onclick={() =>
                  (pickupStatusIds = toggleWorkflowStatusSelection(pickupStatusIds, status.id))}
              >
                {status.name}
              </button>
            {/each}
          </div>
        </div>

        <div class="space-y-2">
          <Label>Finish Statuses</Label>
          <div class="flex flex-wrap gap-2">
            {#each selectableStatuses as status (status.id)}
              <button
                type="button"
                class={cn(
                  'rounded-full border px-3 py-1.5 text-xs transition-colors',
                  finishStatusIds.includes(status.id)
                    ? 'border-primary/40 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground hover:bg-muted',
                )}
                disabled={saving}
                onclick={() =>
                  (finishStatusIds = toggleWorkflowStatusSelection(finishStatusIds, status.id))}
              >
                {status.name}
              </button>
            {/each}
          </div>
        </div>
      </div>

      {#if templateStatusError}
        <p class="text-destructive text-xs">{templateStatusError}</p>
      {/if}
      <Collapsible.Root bind:open={advancedOpen}>
        <Collapsible.Trigger>
          {#snippet child({ props })}
            <button
              {...props}
              type="button"
              class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm transition-colors"
            >
              <ChevronRight class="size-4 transition-transform {advancedOpen ? 'rotate-90' : ''}" />
              Advanced
            </button>
          {/snippet}
        </Collapsible.Trigger>
        <Collapsible.Content>
          <div class="mt-3 space-y-4">
            <div class="space-y-1">
              <div class="text-sm font-medium">Hooks</div>
              <p class="text-muted-foreground text-xs">
                Configure optional workflow and ticket lifecycle hooks.
              </p>
            </div>

            <WorkflowHooksEditor
              draft={hookDraft}
              validation={hookValidation}
              disabled={saving}
              onChange={(nextDraft) => {
                hookDraft = nextDraft
                hookError = ''
              }}
            />

            {#if hookError}
              <p class="text-destructive text-xs">{hookError}</p>
            {/if}
          </div>
        </Collapsible.Content>
      </Collapsible.Root>

      <Dialog.Footer showCloseButton>
        <Button type="submit" disabled={saving || !projectId || !!templateStatusError}>
          {saving ? 'Creating…' : 'Create workflow'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
