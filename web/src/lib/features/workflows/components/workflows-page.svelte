<script lang="ts">
  import { untrack } from 'svelte'
  import { beforeNavigate } from '$app/navigation'
  import { projectPath } from '$lib/stores/app-context'
  import { appStore } from '$lib/stores/app.svelte'
  import { PageScaffold } from '$lib/components/layout'
  import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { ApiError } from '$lib/api/client'
  import {
    bindWorkflowSkills,
    saveWorkflowHarness,
    unbindWorkflowSkills,
    validateHarness,
  } from '$lib/api/openase'
  import type { AgentProvider, HarnessValidationIssue } from '$lib/api/contracts'
  import type {
    HarnessVariableGroup,
    WorkflowAgentOption,
    WorkflowStatusOption,
    WorkflowSummary,
    WorkflowTemplateDraft,
  } from '../types'
  import { normalizeWorkflowType, type SkillState, toHarnessContent } from '../model'
  import { loadWorkflowPageData, loadWorkflowHarness } from '../data'
  import WorkflowsPageBody from './workflows-page-body.svelte'
  import WorkflowsPageHeaderActions from './workflows-page-header-actions.svelte'
  import WorkflowTemplateGallery from './workflow-template-gallery.svelte'
  import type { BuiltinRole } from '$lib/api/contracts'
  const projectAIFocusOwner = 'workflow-page'
  let showDetail = $state(false),
    showCreateDialog = $state(false),
    showTemplateGallery = $state(false),
    showList = $state(true)
  let loading = $state(false),
    loadingHarness = $state(false),
    saving = $state(false),
    validating = $state(false)
  let loadError = $state('')
  let workflows = $state<WorkflowSummary[]>([]),
    selectedId = $state('')
  let harness = $state<ReturnType<typeof toHarnessContent> | null>(null),
    draftHarness = $state('')
  let loadedHarnessWorkflowId = $state('')
  let skillStates = $state<SkillState[]>([])
  let validationIssues = $state<HarnessValidationIssue[]>([])
  let builtinRoleContent = $state(''),
    statuses = $state<WorkflowStatusOption[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let providers = $state<AgentProvider[]>([])
  let variableGroups = $state<HarnessVariableGroup[]>([])
  let selectedWorkflow = $derived(workflows.find((workflow) => workflow.id === selectedId) ?? null)
  let isDirty = $derived(harness ? draftHarness !== harness.rawContent : false)

  // Warn on browser close/refresh with unsaved changes
  $effect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  })

  // Warn on in-app navigation with unsaved changes
  beforeNavigate((navigation) => {
    if (isDirty && !confirm('You have unsaved changes. Are you sure you want to leave?')) {
      navigation.cancel()
    }
  })

  const settingsHref = $derived(
    appStore.currentOrg?.id && appStore.currentProject?.id
      ? projectPath(appStore.currentOrg.id, appStore.currentProject.id, 'settings')
      : null,
  )

  function resetWorkflowContent() {
    workflows = []
    selectedId = ''
    harness = null
    draftHarness = ''
    loadedHarnessWorkflowId = ''
    skillStates = []
    statuses = []
    agentOptions = []
    providers = []
    variableGroups = []
    validationIssues = []
  }

  // Load workflow index when project/org changes (initial page load)
  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      resetWorkflowContent()
      loadError = ''
      loading = false
      return
    }
    // Read selectedId without creating a dependency so this effect
    // does NOT re-run when the user switches between workflows.
    const currentSelectedId = untrack(() => selectedId)
    let cancelled = false
    const load = async () => {
      loading = true
      loadError = ''
      try {
        const payload = await loadWorkflowPageData(projectId, orgId, currentSelectedId)
        if (cancelled) return
        const nextWorkflows = payload.workflows
        workflows = nextWorkflows
        agentOptions = payload.agentOptions
        providers = payload.providers
        if (
          !currentSelectedId ||
          !nextWorkflows.some((workflow) => workflow.id === currentSelectedId) ||
          payload.selectedWorkflowId
        ) {
          selectedId = payload.selectedWorkflowId || nextWorkflows[0]?.id || ''
        }
        skillStates = payload.skillStates
        builtinRoleContent = payload.builtinRoleContent
        statuses = payload.statuses
        variableGroups = payload.variableGroups
        harness = payload.harness
        draftHarness = payload.harness?.rawContent ?? ''
        loadedHarnessWorkflowId = payload.harness && selectedId ? selectedId : ''
        validationIssues = []
      } catch (caughtError) {
        if (cancelled) return
        resetWorkflowContent()
        loadError =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load workflows.'
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

  // Load harness when the selected workflow changes (lightweight, no full-page loading)
  $effect(() => {
    const workflowId = selectedId
    const projectId = appStore.currentProject?.id
    if (!workflowId || !projectId) {
      harness = null
      draftHarness = ''
      loadedHarnessWorkflowId = ''
      validationIssues = []
      return
    }
    if (workflowId === loadedHarnessWorkflowId) return
    let cancelled = false
    const doLoadHarness = async () => {
      loadingHarness = true
      try {
        const payload = await loadWorkflowHarness(projectId, workflowId)
        if (cancelled) return
        harness = payload.harness
        draftHarness = payload.harness.rawContent
        loadedHarnessWorkflowId = workflowId
        validationIssues = []
        skillStates = payload.skillStates
        workflows = workflows.map((workflow) =>
          workflow.id === workflowId ? { ...workflow, history: payload.history } : workflow,
        )
      } catch (caughtError) {
        if (cancelled) return
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load harness.',
        )
      } finally {
        if (!cancelled) {
          loadingHarness = false
        }
      }
    }
    void doLoadHarness()
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId || loading || !selectedWorkflow) {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
      return
    }

    appStore.setProjectAssistantFocus(
      projectAIFocusOwner,
      {
        kind: 'workflow',
        projectId,
        workflowId: selectedWorkflow.id,
        workflowName: selectedWorkflow.name,
        workflowType: selectedWorkflow.type,
        harnessPath: selectedWorkflow.harnessPath,
        isActive: selectedWorkflow.isActive,
        selectedArea: 'harness',
        hasDirtyDraft: isDirty,
      },
      PROJECT_AI_FOCUS_PRIORITY.workspace,
    )

    return () => {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
    }
  })
  async function handleSave() {
    if (!selectedId) return
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    saving = true
    try {
      const payload = await saveWorkflowHarness(selectedId, draftHarness)
      const refreshed = await loadWorkflowHarness(projectId, selectedId)
      harness = refreshed.harness
      draftHarness = refreshed.harness.rawContent
      skillStates = refreshed.skillStates
      workflows = workflows.map((workflow) =>
        workflow.id === selectedId
          ? {
              ...workflow,
              harnessPath: payload.harness.path ?? workflow.harnessPath,
              version: payload.harness.version ?? workflow.version,
              history: refreshed.history,
            }
          : workflow,
      )
      toastStore.success(
        payload.harness.version
          ? `Harness saved as v${payload.harness.version}.`
          : 'Harness saved.',
      )
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save harness.',
      )
    } finally {
      saving = false
    }
  }
  async function handleValidate() {
    validating = true
    try {
      const payload = await validateHarness(draftHarness)
      validationIssues = payload.issues
      if (payload.valid) {
        toastStore.success('Harness is valid.')
      } else {
        toastStore.warning('Harness has validation issues.')
      }
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to validate harness.',
      )
    } finally {
      validating = false
    }
  }
  let templateDraft = $state<WorkflowTemplateDraft | null>(null)

  function handleCreateWorkflow() {
    if (statuses.length === 0 || agentOptions.length === 0) return
    templateDraft = null
    showCreateDialog = true
  }

  function handleUseTemplate(role: BuiltinRole) {
    if (statuses.length === 0 || agentOptions.length === 0) {
      toastStore.error('Configure statuses and agents before creating a workflow.')
      return
    }
    templateDraft = {
      name: role.name,
      content: role.workflow_content || role.content,
      workflowType: normalizeWorkflowType(role.workflow_type),
      harnessPath: role.harness_path,
    }
    showCreateDialog = true
  }
  async function handleToggleSkill(skill: SkillState) {
    if (!selectedId) return
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    if (isDirty) {
      toastStore.warning('Please save your harness changes before binding or unbinding skills.')
      return
    }
    try {
      const result = skill.bound
        ? await unbindWorkflowSkills(selectedId, [skill.name])
        : await bindWorkflowSkills(selectedId, [skill.name])
      const refreshed = await loadWorkflowHarness(projectId, selectedId)
      harness = refreshed.harness
      draftHarness = refreshed.harness.rawContent
      skillStates = refreshed.skillStates
      workflows = workflows.map((workflow) =>
        workflow.id === selectedId
          ? {
              ...workflow,
              harnessPath: result.harness.path ?? workflow.harnessPath,
              version: result.harness.version ?? workflow.version,
              history: refreshed.history,
            }
          : workflow,
      )
      toastStore.success(skill.bound ? `Unbound ${skill.name}.` : `Bound ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow skills.',
      )
    }
  }
  function handleApplyAssistantDraft(content: string) {
    draftHarness = content
    validationIssues = []
    toastStore.info('Applied AI suggestion to the harness draft.')
  }
</script>

{#snippet actions()}
  <WorkflowsPageHeaderActions
    canCreate={statuses.length > 0 && agentOptions.length > 0}
    statusStageHref={settingsHref ? `${settingsHref}#statuses` : null}
    onCreate={handleCreateWorkflow}
    onBrowseTemplates={() => (showTemplateGallery = true)}
  />
{/snippet}

<PageScaffold
  title="Workflows"
  description="Edit published harnesses and manage workflow lifecycle settings."
  variant="workspace"
  {actions}
>
  <WorkflowsPageBody
    {loading}
    {loadingHarness}
    {settingsHref}
    {loadError}
    {workflows}
    {selectedId}
    projectId={appStore.currentProject?.id ?? ''}
    {providers}
    {selectedWorkflow}
    {harness}
    {draftHarness}
    {variableGroups}
    {skillStates}
    {validationIssues}
    {saving}
    {validating}
    {isDirty}
    bind:showDetail
    bind:showCreateDialog
    bind:showList
    {statuses}
    {agentOptions}
    {builtinRoleContent}
    {templateDraft}
    onSelectedIdChange={(id) => {
      if (
        isDirty &&
        !confirm('You have unsaved changes. Are you sure you want to switch workflows?')
      )
        return
      selectedId = id
    }}
    onDraftChange={(raw) => (draftHarness = raw)}
    onApplyAssistantDraft={handleApplyAssistantDraft}
    onSave={() => void handleSave()}
    onValidate={() => void handleValidate()}
    onToggleSkill={(skill) => void handleToggleSkill(skill)}
    onWorkflowsChange={(nextWorkflows) => (workflows = nextWorkflows)}
    onCreated={({ workflow, selectedId: nextSelectedId }) => {
      workflows = [...workflows, workflow]
      selectedId = nextSelectedId
      toastStore.success('Workflow created.')
    }}
  />
</PageScaffold>

<WorkflowTemplateGallery bind:open={showTemplateGallery} onUseTemplate={handleUseTemplate} />
