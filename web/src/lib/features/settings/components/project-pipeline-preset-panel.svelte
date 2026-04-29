<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    AgentPayload,
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
  import { Separator } from '$ui/separator'
  import ProjectPipelinePresetDetail from './project-pipeline-preset-detail.svelte'

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
          <ProjectPipelinePresetDetail
            {selectedPreset}
            {statuses}
            {workflows}
            {agents}
            {agentBindings}
            {applying}
            {presetBlocked}
            {errorMessage}
            {lastAppliedResult}
            {applyDisabled}
            onApply={() => void handleApply()}
            onBindingChange={(workflowKey, agentId) => {
              agentBindings = { ...agentBindings, [workflowKey]: agentId }
            }}
          />
        {/if}
      </div>
    </div>
  {/if}
</div>
