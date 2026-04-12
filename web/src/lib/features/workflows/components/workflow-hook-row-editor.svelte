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
  import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
  import { i18nStore } from '$lib/i18n/store.svelte'

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

  function t(key: TranslationKey, params?: TranslationParams) {
    return i18nStore.t(key, params)
  }

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
        aria-label={t('workflows.hooks.rowEditor.actions.duplicateAria', { title })}
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
        aria-label={t('workflows.hooks.rowEditor.actions.deleteAria', { title })}
        onclick={() => onDelete?.()}
      >
        <Trash2 class="size-3.5" />
      </Button>
    </div>
  </div>

  <div class="space-y-1.5">
    <Label for={`${row.id}-cmd`}>
      {t('workflows.hooks.rowEditor.labels.command')}
    </Label>
    <Input
      id={`${row.id}-cmd`}
      value={row.cmd}
      {disabled}
      placeholder={t('workflows.hooks.rowEditor.placeholders.command')}
      aria-invalid={errors.cmd ? 'true' : undefined}
      oninput={(event) => updateRow({ cmd: (event.currentTarget as HTMLInputElement).value })}
    />
    {#if errors.cmd}
      <p class="text-destructive text-xs">{errors.cmd}</p>
    {/if}
  </div>

  <div class="grid gap-3 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label for={`${row.id}-timeout`}>
        {t('workflows.hooks.rowEditor.labels.timeout')}
      </Label>
      <Input
        id={`${row.id}-timeout`}
        type="number"
        min="0"
        step="1"
        inputmode="numeric"
        value={row.timeout}
        {disabled}
        placeholder={t('workflows.hooks.rowEditor.placeholders.omit')}
        aria-invalid={errors.timeout ? 'true' : undefined}
        oninput={(event) => updateRow({ timeout: (event.currentTarget as HTMLInputElement).value })}
      />
      {#if errors.timeout}
        <p class="text-destructive text-xs">{errors.timeout}</p>
      {/if}
    </div>

    <div class="space-y-1.5">
      <Label>{t('workflows.hooks.rowEditor.labels.onFailure')}</Label>
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
      <Label for={`${row.id}-workdir`}>
        {t('workflows.hooks.rowEditor.labels.workdir')}
      </Label>
      <Input
        id={`${row.id}-workdir`}
        value={row.workdir}
        {disabled}
        placeholder={t('workflows.hooks.rowEditor.placeholders.workdir')}
        oninput={(event) => updateRow({ workdir: (event.currentTarget as HTMLInputElement).value })}
      />
    </div>
  {/if}
</div>
