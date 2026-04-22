<script lang="ts">
  import type {
    AgentPayload,
    PipelinePreset,
    PipelinePresetApplyResult,
    StatusPayload,
    WorkflowListPayload,
  } from '$lib/api/contracts'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'

  type AgentRecord = AgentPayload['agents'][number]
  type StatusRecord = StatusPayload['statuses'][number]
  type WorkflowRecord = WorkflowListPayload['workflows'][number]

  let {
    selectedPreset,
    statuses,
    workflows,
    agents,
    agentBindings,
    applying,
    presetBlocked,
    errorMessage,
    lastAppliedResult,
    applyDisabled,
    onApply,
    onBindingChange,
  }: {
    selectedPreset: PipelinePreset
    statuses: StatusRecord[]
    workflows: WorkflowRecord[]
    agents: AgentRecord[]
    agentBindings: Record<string, string>
    applying: boolean
    presetBlocked: boolean
    errorMessage: string
    lastAppliedResult: PipelinePresetApplyResult | null
    applyDisabled: boolean
    onApply: () => void
    onBindingChange: (workflowKey: string, agentId: string) => void
  } = $props()

  function hasExistingStatus(name: string) {
    return statuses.some((item) => item.name.trim().toLowerCase() === name.trim().toLowerCase())
  }

  function hasExistingWorkflow(name: string) {
    return workflows.some((item) => item.name.trim().toLowerCase() === name.trim().toLowerCase())
  }

  function workflowAgentLabel(agentId: string) {
    return (
      agents.find((item) => item.id === agentId)?.name ??
      i18nStore.t('settings.pipelinePresets.agent.select')
    )
  }
</script>

<div class="border-border space-y-4 rounded-lg border p-4">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <p class="text-foreground text-sm font-semibold">{selectedPreset.preset.name}</p>
      <p class="text-muted-foreground mt-1 text-sm">{selectedPreset.preset.description}</p>
    </div>
    <div class="text-muted-foreground text-xs">
      {i18nStore.t('settings.pipelinePresets.messages.previewSummary', {
        statusCount: selectedPreset.statuses.length,
        workflowCount: selectedPreset.workflows.length,
      })}
    </div>
  </div>

  <div class="grid gap-4 xl:grid-cols-2">
    <div class="space-y-2">
      <Label>{i18nStore.t('settings.pipelinePresets.labels.statusPreview')}</Label>
      <div class="space-y-2">
        {#each selectedPreset.statuses as item (item.name)}
          <div class="bg-muted/30 flex items-center justify-between rounded-md border px-3 py-2 text-sm">
            <div>
              <p class="text-foreground font-medium">{item.name}</p>
              <p class="text-muted-foreground text-xs">{item.stage}</p>
            </div>
            <span class="text-muted-foreground text-xs uppercase">
              {hasExistingStatus(item.name)
                ? i18nStore.t('settings.pipelinePresets.actions.update')
                : i18nStore.t('settings.pipelinePresets.actions.create')}
            </span>
          </div>
        {/each}
      </div>
    </div>

    <div class="space-y-3">
      <Label>{i18nStore.t('settings.pipelinePresets.labels.workflowPreview')}</Label>
      {#each selectedPreset.workflows as item (item.key)}
        <div class="space-y-2 rounded-md border p-3">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-foreground text-sm font-medium">{item.name}</p>
              <p class="text-muted-foreground text-xs">
                {item.pickup_statuses.join(', ')} -> {item.finish_statuses.join(', ')}
              </p>
            </div>
            <span class="text-muted-foreground text-xs uppercase">
              {hasExistingWorkflow(item.name)
                ? i18nStore.t('settings.pipelinePresets.actions.update')
                : i18nStore.t('settings.pipelinePresets.actions.create')}
            </span>
          </div>

          <div class="space-y-1.5">
            <Label>{i18nStore.t('settings.pipelinePresets.labels.boundAgent')}</Label>
            <Select.Root
              type="single"
              value={agentBindings[item.key] ?? ''}
              disabled={agents.length === 0 || applying || presetBlocked}
              onValueChange={(value) => {
                if (!value) return
                onBindingChange(item.key, value)
              }}
            >
              <Select.Trigger class="w-full">
                {workflowAgentLabel(agentBindings[item.key] ?? '')}
              </Select.Trigger>
              <Select.Content>
                {#each agents as agent (agent.id)}
                  <Select.Item value={agent.id}>{agent.name}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
        </div>
      {/each}
    </div>
  </div>

  {#if selectedPreset.project_ai?.skill_references?.length}
    <div class="bg-muted/30 rounded-md border px-3 py-2 text-sm">
      <p class="text-foreground font-medium">
        {i18nStore.t('settings.pipelinePresets.labels.projectAI')}
      </p>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.pipelinePresets.messages.projectAIReserved')}
      </p>
      <div class="mt-2 flex flex-wrap gap-2">
        {#each selectedPreset.project_ai.skill_references as ref (ref.skill)}
          <span class="bg-background rounded-full border px-2 py-1 text-[11px]">{ref.skill}</span>
        {/each}
      </div>
    </div>
  {/if}

  {#if errorMessage}
    <p class="text-destructive text-sm">{errorMessage}</p>
  {/if}

  {#if lastAppliedResult}
    <div class="bg-muted/30 rounded-md border px-3 py-2 text-sm">
      <p class="text-foreground font-medium">
        {i18nStore.t('settings.pipelinePresets.messages.lastAppliedTitle')}
      </p>
      <p class="text-muted-foreground mt-1 text-xs">
        {i18nStore.t('settings.pipelinePresets.messages.lastAppliedDescription', {
          statusCount: lastAppliedResult.statuses.length,
          workflowCount: lastAppliedResult.workflows.length,
        })}
      </p>
    </div>
  {/if}

  <div class="flex items-center justify-end gap-3">
    <Button disabled={applyDisabled} onclick={onApply}>
      {applying
        ? i18nStore.t('settings.pipelinePresets.actions.applying')
        : i18nStore.t('settings.pipelinePresets.actions.apply')}
    </Button>
  </div>
</div>
