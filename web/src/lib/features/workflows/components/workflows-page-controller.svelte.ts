import { untrack } from 'svelte'
import { projectPath } from '$lib/stores/app-context'
import { appStore } from '$lib/stores/app.svelte'
import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat'
import { toastStore } from '$lib/stores/toast.svelte'
import { ApiError } from '$lib/api/client'
import type { AgentProvider, BuiltinRole, HarnessValidationIssue } from '$lib/api/contracts'
import type {
  HarnessVariableGroup,
  ScopeGroup,
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
  WorkflowTemplateDraft,
} from '../types'
import { type SkillState, toHarnessContent } from '../model'
import { getScopeGroups } from '$lib/api/openase'
import { loadWorkflowHarness, loadWorkflowPageData } from '../data'
import {
  type WorkflowsPageControllerActionsState,
  applyAssistantDraft,
  handleCreateWorkflow as openWorkflowCreateDialog,
  handleSaveWorkflow,
  handleToggleWorkflowSkill,
  handleUseWorkflowTemplate,
  handleValidateWorkflow,
  handleWorkflowCreated,
  selectWorkflow,
} from './workflows-page-controller-actions'
import { createWorkflowsPageControllerApi } from './workflows-page-controller-api'
import {
  createWorkflowsPageBeforeUnloadGuard,
  registerWorkflowsPageNavigationGuard,
} from './workflows-page-controller-guards'
import { createResetWorkflowsPageContent } from './workflows-page-controller-reset'
import { createWorkflowsPageControllerState } from './workflows-page-controller-state'

const projectAIFocusOwner = 'workflow-page'
export function createWorkflowsPageController() {
  let showDetail = $state(false),
    showCreateDialog = $state(false),
    showTemplateGallery = $state(false),
    showList = $state(true)
  let loading = $state(false),
    loadingHarness = $state(false),
    saving = $state(false),
    validating = $state(false)
  let loadError = $state(''),
    workflows = $state<WorkflowSummary[]>([]),
    selectedId = $state('')
  let harness = $state<ReturnType<typeof toHarnessContent> | null>(null)
  let draftHarness = $state(''),
    loadedHarnessWorkflowId = $state(''),
    skillStates = $state<SkillState[]>([]),
    validationIssues = $state<HarnessValidationIssue[]>([]),
    builtinRoleContent = $state(''),
    statuses = $state<WorkflowStatusOption[]>([]),
    agentOptions = $state<WorkflowAgentOption[]>([]),
    providers = $state<AgentProvider[]>([]),
    variableGroups = $state<HarnessVariableGroup[]>([]),
    scopeGroups = $state<ScopeGroup[]>([]),
    templateDraft = $state<WorkflowTemplateDraft | null>(null)
  const controllerState: WorkflowsPageControllerActionsState = createWorkflowsPageControllerState({
    getSelectedId: () => selectedId,
    setSelectedId: (value) => (selectedId = value),
    getHarness: () => harness,
    setHarness: (value) => (harness = value),
    getDraftHarness: () => draftHarness,
    setDraftHarness: (value) => (draftHarness = value),
    getLoadedHarnessWorkflowId: () => loadedHarnessWorkflowId,
    setLoadedHarnessWorkflowId: (value) => (loadedHarnessWorkflowId = value),
    getSkillStates: () => skillStates,
    setSkillStates: (value) => (skillStates = value),
    getValidationIssues: () => validationIssues,
    setValidationIssues: (value) => (validationIssues = value),
    getWorkflows: () => workflows,
    setWorkflows: (value) => (workflows = value),
    getTemplateDraft: () => templateDraft,
    setTemplateDraft: (value) => (templateDraft = value),
    getStatuses: () => statuses,
    getAgentOptions: () => agentOptions,
    getIsDirty: () => isDirty,
  })

  const selectedWorkflow = $derived(
    workflows.find((workflow) => workflow.id === selectedId) ?? null,
  )
  const isDirty = $derived(harness ? draftHarness !== harness.rawContent : false)
  const settingsHref = $derived(
    appStore.currentOrg?.id && appStore.currentProject?.id
      ? projectPath(appStore.currentOrg.id, appStore.currentProject.id, 'settings')
      : null,
  )
  $effect(() => createWorkflowsPageBeforeUnloadGuard(isDirty))
  registerWorkflowsPageNavigationGuard(() => isDirty)
  const resetWorkflowContent = createResetWorkflowsPageContent({
    setWorkflows: (value) => (workflows = value),
    setSelectedId: (value) => (selectedId = value),
    setHarness: (value) => (harness = value),
    setDraftHarness: (value) => (draftHarness = value),
    setLoadedHarnessWorkflowId: (value) => (loadedHarnessWorkflowId = value),
    setSkillStates: (value) => (skillStates = value),
    setStatuses: (value) => (statuses = value),
    setAgentOptions: (value) => (agentOptions = value),
    setProviders: (value) => (providers = value),
    setVariableGroups: (value) => (variableGroups = value),
    setValidationIssues: (value) => (validationIssues = value),
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      resetWorkflowContent()
      loadError = ''
      loading = false
      return
    }

    const currentSelectedId = untrack(() => selectedId)
    let cancelled = false
    const load = async () => {
      loading = true
      loadError = ''
      try {
        const [payload, fetchedScopeGroups] = await Promise.all([
          loadWorkflowPageData(projectId, orgId, currentSelectedId),
          getScopeGroups(projectId).catch(() => []),
        ])
        if (cancelled) return

        scopeGroups = fetchedScopeGroups
        workflows = payload.workflows
        agentOptions = payload.agentOptions
        providers = payload.providers
        if (
          !currentSelectedId ||
          !payload.workflows.some((workflow) => workflow.id === currentSelectedId) ||
          payload.selectedWorkflowId
        ) {
          selectedId = payload.selectedWorkflowId || payload.workflows[0]?.id || ''
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
        if (!cancelled) loading = false
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
        if (!cancelled) loadingHarness = false
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

  return createWorkflowsPageControllerApi({
    getShowDetail: () => showDetail,
    setShowDetail: (value) => (showDetail = value),
    getShowCreateDialog: () => showCreateDialog,
    setShowCreateDialog: (value) => (showCreateDialog = value),
    getShowTemplateGallery: () => showTemplateGallery,
    setShowTemplateGallery: (value) => (showTemplateGallery = value),
    getShowList: () => showList,
    setShowList: (value) => (showList = value),
    getLoading: () => loading,
    getLoadingHarness: () => loadingHarness,
    getSaving: () => saving,
    getValidating: () => validating,
    getLoadError: () => loadError,
    getWorkflows: () => workflows,
    setWorkflows: (value) => (workflows = value),
    getSelectedId: () => selectedId,
    getHarness: () => harness,
    getDraftHarness: () => draftHarness,
    setDraftHarness: (value) => (draftHarness = value),
    getSkillStates: () => skillStates,
    getValidationIssues: () => validationIssues,
    getBuiltinRoleContent: () => builtinRoleContent,
    getStatuses: () => statuses,
    getAgentOptions: () => agentOptions,
    getProviders: () => providers,
    getVariableGroups: () => variableGroups,
    getScopeGroups: () => scopeGroups,
    getSelectedWorkflow: () => selectedWorkflow,
    getIsDirty: () => isDirty,
    getSettingsHref: () => settingsHref,
    getTemplateDraft: () => templateDraft,
    handleCreateWorkflow: () => {
      showCreateDialog = openWorkflowCreateDialog(controllerState)
    },
    handleUseTemplate: (role: BuiltinRole) => {
      showCreateDialog = handleUseWorkflowTemplate(controllerState, role)
    },
    handleSelectWorkflow: (nextSelectedId: string) => {
      selectWorkflow(controllerState, nextSelectedId)
    },
    handleApplyAssistantDraft: (content: string) => {
      applyAssistantDraft(controllerState, content)
    },
    handleSave: async () => {
      saving = true
      try {
        await handleSaveWorkflow(controllerState)
      } finally {
        saving = false
      }
    },
    handleValidate: async () => {
      validating = true
      try {
        await handleValidateWorkflow(controllerState)
      } finally {
        validating = false
      }
    },
    handleToggleSkill: (skill: SkillState) => handleToggleWorkflowSkill(controllerState, skill),
    handleCreated: (input: { workflow: WorkflowSummary; selectedId: string }) => {
      handleWorkflowCreated(controllerState, input)
    },
  })
}
