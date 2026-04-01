<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Copy, Trash2 } from '@lucide/svelte'
  import {
    listWorkflowHookFailurePolicies,
    type WorkflowHookFailurePolicy,
    type WorkflowHookRowDraft,
    type WorkflowHookRowDraftErrors,
  } from '../workflow-hooks'

  let {
    row,
    title,
    errors = {},
    allowWorkdir = false,
    disabled = false,
    onChange,
    onDuplicate,
    onDelete,
  }: {
    row: WorkflowHookRowDraft
    title: string
    errors?: WorkflowHookRowDraftErrors
    allowWorkdir?: boolean
    disabled?: boolean
    onChange?: (row: WorkflowHookRowDraft) => void
    onDuplicate?: () => void
    onDelete?: () => void
  } = $props()

  const failurePolicies = listWorkflowHookFailurePolicies()
  const failurePolicyLabel = $derived(
    row.onFailure
      .split('_')
      .map((part) => part[0]?.toUpperCase() + part.slice(1))
      .join(' '),
  )

  function updateRow(patch: Partial<WorkflowHookRowDraft>) {
    onChange?.({
      ...row,
      ...patch,
    })
  }
</script>

<div class="bg-background space-y-3 rounded-xl border p-3">
  <div class="flex items-center justify-between gap-2">
    <div class="text-sm font-medium">{title}</div>
    <div class="flex items-center gap-1">
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        {disabled}
        aria-label={`Duplicate ${title}`}
        onclick={() => onDuplicate?.()}
      >
        <Copy class="size-3.5" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        class="text-destructive hover:text-destructive"
        {disabled}
        aria-label={`Delete ${title}`}
        onclick={() => onDelete?.()}
      >
        <Trash2 class="size-3.5" />
      </Button>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for={`${row.id}-cmd`}>Command</Label>
    <Input
      id={`${row.id}-cmd`}
      value={row.cmd}
      {disabled}
      placeholder="bash scripts/ci/run-tests.sh"
      aria-invalid={errors.cmd ? 'true' : undefined}
      oninput={(event) => updateRow({ cmd: (event.currentTarget as HTMLInputElement).value })}
    />
    {#if errors.cmd}
      <p class="text-destructive text-xs">{errors.cmd}</p>
    {/if}
  </div>

  <div class="grid gap-3 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for={`${row.id}-timeout`}>Timeout (sec)</Label>
      <Input
        id={`${row.id}-timeout`}
        type="number"
        min="0"
        step="1"
        inputmode="numeric"
        value={row.timeout}
        {disabled}
        placeholder="Omit"
        aria-invalid={errors.timeout ? 'true' : undefined}
        oninput={(event) => updateRow({ timeout: (event.currentTarget as HTMLInputElement).value })}
      />
      {#if errors.timeout}
        <p class="text-destructive text-xs">{errors.timeout}</p>
      {/if}
    </div>

    <div class="space-y-1.5">
      <Label>On Failure</Label>
      <Select.Root
        type="single"
        value={row.onFailure}
        {disabled}
        onValueChange={(value) =>
          updateRow({ onFailure: (value || 'block') as WorkflowHookFailurePolicy })}
      >
        <Select.Trigger class="w-full">{failurePolicyLabel}</Select.Trigger>
        <Select.Content>
          {#each failurePolicies as policy (policy)}
            <Select.Item value={policy}>
              {policy[0]?.toUpperCase() + policy.slice(1)}
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      {#if errors.onFailure}
        <p class="text-destructive text-xs">{errors.onFailure}</p>
      {/if}
    </div>
  </div>

  {#if allowWorkdir}
    <div class="space-y-1.5">
      <Label for={`${row.id}-workdir`}>Workdir</Label>
      <Input
        id={`${row.id}-workdir`}
        value={row.workdir}
        {disabled}
        placeholder="frontend"
        oninput={(event) => updateRow({ workdir: (event.currentTarget as HTMLInputElement).value })}
      />
    </div>
  {/if}
</div>
