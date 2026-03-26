<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ApiError } from '$lib/api/client'
  import {
    bindWorkflowSkills,
    saveWorkflowHarness,
    unbindWorkflowSkills,
    validateHarness,
  } from '$lib/api/openase'
  import type { HarnessValidationIssue } from '$lib/api/contracts'
  import type {
    HarnessVariableGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
  } from '../types'
  import { type SkillState, toHarnessContent } from '../model'
  import { loadWorkflowHarness, loadWorkflowIndex } from '../data'
  import WorkflowList from './workflow-list.svelte'
  import WorkflowCreationDialog from './workflow-creation-dialog.svelte'
  import WorkflowEditorPanel from './workflow-editor-panel.svelte'
  import WorkflowLifecycleSidebar from './workflow-lifecycle-sidebar.svelte'
  import WorkflowsPageToolbar from './workflows-page-toolbar.svelte'

  let showDetail = $state(true)
  let showCreateDialog = $state(false)
  let loading = $state(false),
    saving = $state(false),
    validating = $state(false)
  let error = $state(''),
    statusMessage = $state('')
  let workflows = $state<WorkflowSummary[]>([]),
    selectedId = $state('')
  let harness = $state<ReturnType<typeof toHarnessContent> | null>(null),
    draftHarness = $state('')
  let skillStates = $state<SkillState[]>([])
  let validationIssues = $state<HarnessValidationIssue[]>([])
  let builtinRoleContent = $state(''),
    statuses = $state<WorkflowStatusOption[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let variableGroups = $state<HarnessVariableGroup[]>([])
  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)
  let isDirty = $derived(harness ? draftHarness !== harness.rawContent : false)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      workflows = []
      selectedId = ''
      harness = null
      draftHarness = ''
      skillStates = []
      statuses = []
      agentOptions = []
      variableGroups = []
      validationIssues = []
      statusMessage = ''
      error = ''
      loading = false
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await loadWorkflowIndex(projectId, orgId, selectedId)
        if (cancelled) return

        const nextWorkflows = payload.workflows
        workflows = nextWorkflows
        agentOptions = payload.agentOptions
        if (!selectedId || !nextWorkflows.some((workflow) => workflow.id === selectedId)) {
          selectedId = nextWorkflows[0]?.id ?? ''
        }

        skillStates = payload.skillStates
        builtinRoleContent = payload.builtinRoleContent
        statuses = payload.statuses
        variableGroups = payload.variableGroups
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load workflows.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const workflowId = selectedId
    const projectId = appStore.currentProject?.id
    if (!workflowId || !projectId) {
      harness = null
      draftHarness = ''
      validationIssues = []
      return
    }

    let cancelled = false

    const loadHarness = async () => {
      try {
        const payload = await loadWorkflowHarness(projectId, workflowId)
        if (cancelled) return

        harness = payload.harness
        draftHarness = payload.harness.rawContent
        validationIssues = []
        skillStates = payload.skillStates
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load harness.'
      }
    }

    void loadHarness()

    return () => {
      cancelled = true
    }
  })

  async function handleSave() {
    if (!selectedId) return

    saving = true
    statusMessage = ''
    error = ''

    try {
      const payload = await saveWorkflowHarness(selectedId, draftHarness)
      harness = toHarnessContent(payload.harness.content)
      draftHarness = payload.harness.content
      workflows = workflows.map((workflow) =>
        workflow.id === selectedId
          ? {
              ...workflow,
              harnessPath: payload.harness.path ?? workflow.harnessPath,
              version: payload.harness.version ?? workflow.version,
            }
          : workflow,
      )
      statusMessage = payload.harness.version
        ? `Harness saved as v${payload.harness.version}.`
        : 'Harness saved.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save harness.'
    } finally {
      saving = false
    }
  }

  async function handleValidate() {
    validating = true
    statusMessage = ''
    error = ''

    try {
      const payload = await validateHarness(draftHarness)
      validationIssues = payload.issues
      statusMessage = payload.valid ? 'Harness is valid.' : 'Harness has validation issues.'
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to validate harness.'
    } finally {
      validating = false
    }
  }

  async function handleCreateWorkflow() {
    if (statuses.length === 0 || agentOptions.length === 0) return
    showCreateDialog = true
  }

  async function handleToggleSkill(skill: SkillState) {
    if (!selectedId) return

    error = ''
    statusMessage = ''

    try {
      if (skill.bound) {
        await unbindWorkflowSkills(selectedId, [skill.path])
        skillStates = skillStates.map((item) =>
          item.path === skill.path ? { ...item, bound: false } : item,
        )
        statusMessage = `Unbound ${skill.name}.`
        return
      }

      await bindWorkflowSkills(selectedId, [skill.path])
      skillStates = skillStates.map((item) =>
        item.path === skill.path ? { ...item, bound: true } : item,
      )
      statusMessage = `Bound ${skill.name}.`
    } catch (caughtError) {
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow skills.'
    }
  }

  function handleApplyAssistantDraft(content: string) {
    draftHarness = content
    validationIssues = []
    error = ''
    statusMessage = 'Applied AI suggestion to the harness draft.'
  }
</script>

<div class="flex h-full flex-col">
  <WorkflowsPageToolbar
    {showDetail}
    canCreate={statuses.length > 0 && agentOptions.length > 0}
    onToggleDetail={() => (showDetail = !showDetail)}
    onCreate={handleCreateWorkflow}
  />
  {#if loading}
    <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
      Loading workflows…
    </div>
  {:else if error && workflows.length === 0}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <div class="flex flex-1 overflow-hidden">
      <div class="w-60 shrink-0">
        <WorkflowList {workflows} {selectedId} onselect={(id) => (selectedId = id)} />
      </div>

      <WorkflowEditorPanel
        projectId={appStore.currentProject?.id}
        selectedWorkflow={selectedWorkflow ?? undefined}
        harness={harness ? toHarnessContent(draftHarness) : null}
        {variableGroups}
        {skillStates}
        {validationIssues}
        {statusMessage}
        {error}
        {saving}
        {validating}
        {isDirty}
        onDraftChange={(raw) => (draftHarness = raw)}
        onApplyAssistantDraft={handleApplyAssistantDraft}
        onSave={() => void handleSave()}
        onValidate={() => void handleValidate()}
        onToggleSkill={(skill) => void handleToggleSkill(skill)}
      />

      {#if showDetail && selectedWorkflow}
        <div class="w-70 shrink-0">
          <WorkflowLifecycleSidebar
            workflow={selectedWorkflow}
            {workflows}
            {statuses}
            {agentOptions}
            onWorkflowsChange={(nextWorkflows) => (workflows = nextWorkflows)}
            onSelectedIdChange={(nextSelectedId) => (selectedId = nextSelectedId)}
          />
        </div>
      {/if}
    </div>
  {/if}
</div>

<WorkflowCreationDialog
  bind:open={showCreateDialog}
  projectId={appStore.currentProject?.id ?? ''}
  {statuses}
  {agentOptions}
  existingCount={workflows.length}
  {builtinRoleContent}
  onCreated={({ workflow, selectedId: nextSelectedId }) => {
    workflows = [...workflows, workflow]
    selectedId = nextSelectedId
    statusMessage = 'Workflow created.'
    error = ''
  }}
/>
