<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    AgentPayload,
    PipelinePreset,
    PipelinePresetApplyResult,
    PipelinePresetCatalog,
    StatusPayload,
    WorkflowListPayload,
  } from '$lib/api/contracts'
  import {
    applyPipelinePreset,
    listAgents,
    listPipelinePresets,
    listStatuses,
    listWorkflows,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import * as Select from '$ui/select'

  type AgentRecord = AgentPayload['agents'][number]
  type StatusRecord = StatusPayload['statuses'][number]
  type WorkflowRecord = WorkflowListPayload['workflows'][number]

  let loading = $state(false)
  let applying = $state(false)
  let errorMessage = $state('')
  let catalog = $state<PipelinePresetCatalog | null>(null)
  let agents = $state<AgentRecord[]>([])
  let statuses = $state<StatusRecord[]>([])
  let workflows = $state<WorkflowRecord[]>([])
  let selectedPresetKey = $state('')
  let agentBindings = $state<Record<string, string>>({})
  let lastAppliedResult = $state<PipelinePresetApplyResult | null>(null)

  const selectedPreset = $derived(
    catalog?.presets.find((item) => item.preset.key === selectedPresetKey) ?? null,
  )
  const activeTicketCount = $derived(catalog?.active_ticket_count ?? 0)
  const presetBlocked = $derived(activeTicketCount > 0)
  const missingAgentBindingCount = $derived.by(() => {
    if (!selectedPreset) return 0
    return selectedPreset.workflows.filter((item) => !agentBindings[item.key]).length
  })
  const applyDisabled = $derived(
    applying ||
      loading ||
      !selectedPreset ||
      presetBlocked ||
      agents.length === 0 ||
      missingAgentBindingCount > 0,
  )

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

  async function loadData(projectId: string) {
    loading = true
    errorMessage = ''
    try {
      const [nextCatalog, nextAgents, nextStatuses, nextWorkflows] = await Promise.all([
        listPipelinePresets(projectId),
        listAgents(projectId),
        listStatuses(projectId),
        listWorkflows(projectId),
      ])
      catalog = nextCatalog
      agents = nextAgents.agents ?? []
      statuses = nextStatuses.statuses ?? []
      workflows = nextWorkflows.workflows ?? []
      if (
        !selectedPresetKey ||
        !nextCatalog.presets.some((item) => item.preset.key === selectedPresetKey)
      ) {
        selectedPresetKey = nextCatalog.presets[0]?.preset.key ?? ''
      }
      if (nextCatalog.presets.length === 0) {
        agentBindings = {}
      }
    } catch (caughtError) {
      errorMessage =
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.pipelinePresets.errors.load')
    } finally {
      loading = false
    }
  }

  async function handleApply() {
    const projectId = appStore.currentProject?.id
    const preset = selectedPreset
    if (!projectId || !preset) return

    applying = true
    errorMessage = ''
    try {
      const payload = await applyPipelinePreset(projectId, preset.preset.key, {
        workflow_agent_bindings: preset.workflows.map((item) => ({
          workflow_key: item.key,
          agent_id: agentBindings[item.key],
        })),
      })
      lastAppliedResult = payload.result
      await loadData(projectId)
      toastStore.success(
        i18nStore.t('settings.pipelinePresets.messages.applySuccess', {
          presetName: preset.preset.name,
        }),
      )
    } catch (caughtError) {
      const message =
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('settings.pipelinePresets.errors.apply')
      errorMessage = message
      toastStore.error(message)
    } finally {
      applying = false
    }
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      catalog = null
      agents = []
      statuses = []
      workflows = []
      selectedPresetKey = ''
      agentBindings = {}
      lastAppliedResult = null
      errorMessage = ''
      loading = false
      applying = false
      return
    }

    let cancelled = false
    void loadData(projectId).then(() => {
      if (cancelled) return
    })
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const preset = selectedPreset
    if (!preset) {
      if (Object.keys(agentBindings).length > 0) {
        agentBindings = {}
      }
      return
    }
    const nextBindings = { ...agentBindings }
    const fallbackAgentId = agents[0]?.id ?? ''
    let changed = false
    for (const workflow of preset.workflows) {
      if (!nextBindings[workflow.key] && fallbackAgentId) {
        nextBindings[workflow.key] = fallbackAgentId
        changed = true
      }
    }
    for (const key of Object.keys(nextBindings)) {
      if (!preset.workflows.some((item) => item.key === key)) {
        delete nextBindings[key]
        changed = true
      }
    }
    if (changed) {
      agentBindings = nextBindings
    }
  })
</script>

<div class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('settings.pipelinePresets.heading')}
    </h3>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.pipelinePresets.description')}
    </p>
  </div>

  <Separator />

  {#if loading && !catalog}
    <div class="text-muted-foreground text-sm">
      {i18nStore.t('settings.pipelinePresets.messages.loading')}
    </div>
  {:else if errorMessage && !catalog}
    <div class="text-destructive text-sm">{errorMessage}</div>
  {:else if catalog?.presets.length === 0}
    <div class="text-muted-foreground text-sm">
      {i18nStore.t('settings.pipelinePresets.messages.empty')}
    </div>
  {:else}
    <div class="space-y-4">
      {#if presetBlocked}
        <div class="bg-muted/40 border-border rounded-lg border px-4 py-3 text-sm">
          <p class="text-foreground font-medium">
            {i18nStore.t('settings.pipelinePresets.messages.blockedTitle')}
          </p>
          <p class="text-muted-foreground mt-1">
            {i18nStore.t('settings.pipelinePresets.messages.blockedDescription', {
              count: activeTicketCount,
            })}
          </p>
        </div>
      {/if}

      {#if agents.length === 0}
        <div class="bg-muted/40 border-border rounded-lg border px-4 py-3 text-sm">
          <p class="text-foreground font-medium">
            {i18nStore.t('settings.pipelinePresets.messages.noAgentsTitle')}
          </p>
          <p class="text-muted-foreground mt-1">
            {i18nStore.t('settings.pipelinePresets.messages.noAgentsDescription')}
          </p>
        </div>
      {/if}

      <div class="grid gap-3 lg:grid-cols-[minmax(0,16rem)_minmax(0,1fr)]">
        <div class="space-y-2">
          {#each catalog?.presets ?? [] as preset (preset.preset.key)}
            <button
              type="button"
              class={selectedPresetKey === preset.preset.key
                ? 'border-foreground bg-muted text-foreground rounded-lg border p-3 text-left transition-colors'
                : 'border-border hover:bg-muted/50 text-foreground rounded-lg border p-3 text-left transition-colors'}
              onclick={() => {
                selectedPresetKey = preset.preset.key
                lastAppliedResult = null
              }}
            >
              <p class="text-sm font-semibold">{preset.preset.name}</p>
              <p class="text-muted-foreground mt-1 text-xs leading-5">
                {preset.preset.description}
              </p>
            </button>
          {/each}
        </div>

        {#if selectedPreset}
          <div class="border-border space-y-4 rounded-lg border p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p class="text-foreground text-sm font-semibold">{selectedPreset.preset.name}</p>
                <p class="text-muted-foreground mt-1 text-sm">
                  {selectedPreset.preset.description}
                </p>
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
                    <div
                      class="bg-muted/30 flex items-center justify-between rounded-md border px-3 py-2 text-sm"
                    >
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
                          agentBindings = { ...agentBindings, [item.key]: value }
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
                    <span class="bg-background rounded-full border px-2 py-1 text-[11px]">
                      {ref.skill}
                    </span>
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
              <Button disabled={applyDisabled} onclick={() => void handleApply()}>
                {applying
                  ? i18nStore.t('settings.pipelinePresets.actions.applying')
                  : i18nStore.t('settings.pipelinePresets.actions.apply')}
              </Button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
