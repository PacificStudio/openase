import type { AgentProvider, BuiltinRole, HarnessValidationIssue } from '$lib/api/contracts'
import type {
  HarnessVariableGroup,
  ScopeGroup,
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
  WorkflowTemplateDraft,
} from '../types'
import type { SkillState } from '../model'

type WorkflowsPageControllerApiInput = {
  getShowDetail: () => boolean
  setShowDetail: (value: boolean) => void
  getShowCreateDialog: () => boolean
  setShowCreateDialog: (value: boolean) => void
  getShowTemplateGallery: () => boolean
  setShowTemplateGallery: (value: boolean) => void
  getShowList: () => boolean
  setShowList: (value: boolean) => void
  getLoading: () => boolean
  getLoadingHarness: () => boolean
  getSaving: () => boolean
  getValidating: () => boolean
  getLoadError: () => string
  getWorkflows: () => WorkflowSummary[]
  setWorkflows: (value: WorkflowSummary[]) => void
  getSelectedId: () => string
  getHarness: () => ReturnType<typeof import('../model').toHarnessContent> | null
  getDraftHarness: () => string
  setDraftHarness: (value: string) => void
  getSkillStates: () => SkillState[]
  getValidationIssues: () => HarnessValidationIssue[]
  getBuiltinRoleContent: () => string
  getStatuses: () => WorkflowStatusOption[]
  getAgentOptions: () => WorkflowAgentOption[]
  getProviders: () => AgentProvider[]
  getVariableGroups: () => HarnessVariableGroup[]
  getScopeGroups: () => ScopeGroup[]
  getSelectedWorkflow: () => WorkflowSummary | null
  getIsDirty: () => boolean
  getSettingsHref: () => string | null
  getTemplateDraft: () => WorkflowTemplateDraft | null
  handleCreateWorkflow: () => void
  handleUseTemplate: (role: BuiltinRole) => void
  handleSelectWorkflow: (nextSelectedId: string) => void
  handleApplyAssistantDraft: (content: string) => void
  handleSave: () => Promise<void>
  handleValidate: () => Promise<void>
  handleToggleSkill: (skill: SkillState) => Promise<void> | void
  handleCreated: (input: { workflow: WorkflowSummary; selectedId: string }) => void
}

export function createWorkflowsPageControllerApi(input: WorkflowsPageControllerApiInput) {
  return {
    get showDetail() {
      return input.getShowDetail()
    },
    set showDetail(value: boolean) {
      input.setShowDetail(value)
    },
    get showCreateDialog() {
      return input.getShowCreateDialog()
    },
    set showCreateDialog(value: boolean) {
      input.setShowCreateDialog(value)
    },
    get showTemplateGallery() {
      return input.getShowTemplateGallery()
    },
    set showTemplateGallery(value: boolean) {
      input.setShowTemplateGallery(value)
    },
    get showList() {
      return input.getShowList()
    },
    set showList(value: boolean) {
      input.setShowList(value)
    },
    get loading() {
      return input.getLoading()
    },
    get loadingHarness() {
      return input.getLoadingHarness()
    },
    get saving() {
      return input.getSaving()
    },
    get validating() {
      return input.getValidating()
    },
    get loadError() {
      return input.getLoadError()
    },
    get workflows() {
      return input.getWorkflows()
    },
    set workflows(value: WorkflowSummary[]) {
      input.setWorkflows(value)
    },
    get selectedId() {
      return input.getSelectedId()
    },
    get harness() {
      return input.getHarness()
    },
    get draftHarness() {
      return input.getDraftHarness()
    },
    set draftHarness(value: string) {
      input.setDraftHarness(value)
    },
    get skillStates() {
      return input.getSkillStates()
    },
    get validationIssues() {
      return input.getValidationIssues()
    },
    get builtinRoleContent() {
      return input.getBuiltinRoleContent()
    },
    get statuses() {
      return input.getStatuses()
    },
    get agentOptions() {
      return input.getAgentOptions()
    },
    get providers() {
      return input.getProviders()
    },
    get variableGroups() {
      return input.getVariableGroups()
    },
    get scopeGroups() {
      return input.getScopeGroups()
    },
    get selectedWorkflow() {
      return input.getSelectedWorkflow()
    },
    get isDirty() {
      return input.getIsDirty()
    },
    get settingsHref() {
      return input.getSettingsHref()
    },
    get templateDraft() {
      return input.getTemplateDraft()
    },
    handleCreateWorkflow: input.handleCreateWorkflow,
    handleUseTemplate: input.handleUseTemplate,
    handleSelectWorkflow: input.handleSelectWorkflow,
    handleApplyAssistantDraft: input.handleApplyAssistantDraft,
    handleSave: input.handleSave,
    handleValidate: input.handleValidate,
    handleToggleSkill: input.handleToggleSkill,
    handleCreated: input.handleCreated,
  }
}
