import { untrack } from 'svelte'
import { beforeNavigate } from '$app/navigation'
import { projectPath } from '$lib/stores/app-context'
import { appStore } from '$lib/stores/app.svelte'
import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat'
import { toastStore } from '$lib/stores/toast.svelte'
import { ApiError } from '$lib/api/client'
import type { AgentProvider, BuiltinRole, HarnessValidationIssue } from '$lib/api/contracts'
import type {
  HarnessVariableGroup,
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
  WorkflowTemplateDraft,
} from '../types'
import { type SkillState, toHarnessContent } from '../model'
import { loadWorkflowHarness, loadWorkflowPageData } from '../data'
import {
  createWorkflowsPageControllerActionsState,
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

const projectAIFocusOwner = 'workflow-page'

export function createWorkflowsPageController() {
  let showDetail = $state(false)
  let showCreateDialog = $state(false)
  let showTemplateGallery = $state(false)
  let showList = $state(true)
  let loading = $state(false)
  let loadingHarness = $state(false)
  let saving = $state(false)
  let validating = $state(false)
  let loadError = $state('')
  let workflows = $state<WorkflowSummary[]>([])
  let selectedId = $state('')
  let harness = $state<ReturnType<typeof toHarnessContent> | null>(null)
  let draftHarness = $state('')
  let loadedHarnessWorkflowId = $state('')
  let skillStates = $state<SkillState[]>([])
  let validationIssues = $state<HarnessValidationIssue[]>([])
  let builtinRoleContent = $state('')
  let statuses = $state<WorkflowStatusOption[]>([])
  let agentOptions = $state<WorkflowAgentOption[]>([])
  let providers = $state<AgentProvider[]>([])
  let variableGroups = $state<HarnessVariableGroup[]>([])
  let templateDraft = $state<WorkflowTemplateDraft | null>(null)
  const controllerState: WorkflowsPageControllerActionsState =
    createWorkflowsPageControllerActionsState({
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

  $effect(() => {
    if (!isDirty) return
    const handler = (event: BeforeUnloadEvent) => {
      event.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  })

  beforeNavigate((navigation) => {
    if (isDirty && !confirm('You have unsaved changes. Are you sure you want to leave?'))
      navigation.cancel()
  })

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
        const payload = await loadWorkflowPageData(projectId, orgId, currentSelectedId)
        if (cancelled) return

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

  return {
    get showDetail() {
      return showDetail
    },
    set showDetail(value: boolean) {
      showDetail = value
    },
    get showCreateDialog() {
      return showCreateDialog
    },
    set showCreateDialog(value: boolean) {
      showCreateDialog = value
    },
    get showTemplateGallery() {
      return showTemplateGallery
    },
    set showTemplateGallery(value: boolean) {
      showTemplateGallery = value
    },
    get showList() {
      return showList
    },
    set showList(value: boolean) {
      showList = value
    },
    get loading() {
      return loading
    },
    get loadingHarness() {
      return loadingHarness
    },
    get saving() {
      return saving
    },
    get validating() {
      return validating
    },
    get loadError() {
      return loadError
    },
    get workflows() {
      return workflows
    },
    set workflows(value: WorkflowSummary[]) {
      workflows = value
    },
    get selectedId() {
      return selectedId
    },
    get harness() {
      return harness
    },
    get draftHarness() {
      return draftHarness
    },
    set draftHarness(value: string) {
      draftHarness = value
    },
    get skillStates() {
      return skillStates
    },
    get validationIssues() {
      return validationIssues
    },
    get builtinRoleContent() {
      return builtinRoleContent
    },
    get statuses() {
      return statuses
    },
    get agentOptions() {
      return agentOptions
    },
    get providers() {
      return providers
    },
    get variableGroups() {
      return variableGroups
    },
    get selectedWorkflow() {
      return selectedWorkflow
    },
    get isDirty() {
      return isDirty
    },
    get settingsHref() {
      return settingsHref
    },
    get templateDraft() {
      return templateDraft
    },
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
  }
}
