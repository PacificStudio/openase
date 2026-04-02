type WorkflowsPageResetInput = {
  setWorkflows: (value: import('../types').WorkflowSummary[]) => void
  setSelectedId: (value: string) => void
  setHarness: (value: ReturnType<typeof import('../model').toHarnessContent> | null) => void
  setDraftHarness: (value: string) => void
  setLoadedHarnessWorkflowId: (value: string) => void
  setSkillStates: (value: import('../model').SkillState[]) => void
  setStatuses: (value: import('../types').WorkflowStatusOption[]) => void
  setAgentOptions: (value: import('../types').WorkflowAgentOption[]) => void
  setProviders: (value: import('$lib/api/contracts').AgentProvider[]) => void
  setVariableGroups: (value: import('../types').HarnessVariableGroup[]) => void
  setValidationIssues: (value: import('$lib/api/contracts').HarnessValidationIssue[]) => void
}

export function createResetWorkflowsPageContent(input: WorkflowsPageResetInput) {
  return () => {
    input.setWorkflows([])
    input.setSelectedId('')
    input.setHarness(null)
    input.setDraftHarness('')
    input.setLoadedHarnessWorkflowId('')
    input.setSkillStates([])
    input.setStatuses([])
    input.setAgentOptions([])
    input.setProviders([])
    input.setVariableGroups([])
    input.setValidationIssues([])
  }
}
