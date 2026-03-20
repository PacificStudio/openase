import { api, toErrorMessage } from './api'
import type {
  HarnessPayload,
  HarnessValidationIssue,
  HarnessValidationResponse,
  Workflow,
} from './types'

type HarnessState = {
  harnessBusy: boolean
  validationBusy: boolean
  errorMessage: string
  notice: string
  selectedWorkflowId: string
  selectedWorkflow: Workflow | null
  workflows: Workflow[]
  harnessDraft: string
  harnessPath: string
  harnessVersion: number
  harnessIssues: HarnessValidationIssue[]
}

export function createHarnessActions(state: HarnessState) {
  let validationRunID = 0
  let validationTimer: ReturnType<typeof setTimeout> | null = null

  function harnessDirty() {
    return Boolean(
      state.selectedWorkflow &&
      state.harnessDraft !== (state.selectedWorkflow.harness_content ?? ''),
    )
  }

  function setHarnessDraft(value: string, validate = true) {
    state.harnessDraft = value
    if (!state.selectedWorkflowId) {
      state.validationBusy = false
      state.harnessIssues = []
      clearPendingValidation()
      return
    }
    if (validate) {
      queueHarnessValidation(value)
    }
  }

  async function validateHarnessNow() {
    clearPendingValidation()
    state.errorMessage = ''
    await runHarnessValidation(state.harnessDraft)
  }

  async function saveHarness() {
    const selectedWorkflow = state.selectedWorkflow
    if (!selectedWorkflow) {
      return
    }

    state.harnessBusy = true
    state.errorMessage = ''
    state.notice = ''
    try {
      clearPendingValidation()
      const valid = await runHarnessValidation(state.harnessDraft)
      if (!valid) {
        state.errorMessage = 'Harness validation failed. Resolve YAML errors before saving.'
        return
      }

      const payload = await api<HarnessPayload>(
        `/api/v1/workflows/${selectedWorkflow.id}/harness`,
        {
          method: 'PUT',
          body: JSON.stringify({ content: state.harnessDraft }),
        },
      )
      state.harnessPath = payload.harness.path
      state.harnessVersion = payload.harness.version
      state.selectedWorkflow = {
        ...selectedWorkflow,
        harness_content: state.harnessDraft,
        harness_path: payload.harness.path,
        version: payload.harness.version,
      }
      state.workflows = state.workflows.map((item) =>
        item.id === selectedWorkflow.id
          ? { ...item, harness_path: payload.harness.path, version: payload.harness.version }
          : item,
      )
      state.notice = `Harness saved for ${selectedWorkflow.name}`
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.harnessBusy = false
    }
  }

  function destroy() {
    clearPendingValidation()
  }

  return {
    harnessDirty,
    setHarnessDraft,
    validateHarnessNow,
    saveHarness,
    destroy,
  }

  async function runHarnessValidation(content: string, runID = ++validationRunID) {
    state.validationBusy = true
    try {
      const response = await api<HarnessValidationResponse>('/api/v1/harness/validate', {
        method: 'POST',
        body: JSON.stringify({ content }),
      })
      if (runID === validationRunID) {
        state.harnessIssues = response.issues
      }
      return response.valid
    } catch (error) {
      if (runID === validationRunID) {
        state.harnessIssues = [{ level: 'error', message: toErrorMessage(error) }]
      }
      return false
    } finally {
      if (runID === validationRunID) {
        state.validationBusy = false
      }
    }
  }

  function queueHarnessValidation(content: string) {
    clearPendingValidation()
    state.validationBusy = true
    const runID = ++validationRunID
    validationTimer = setTimeout(() => void runHarnessValidation(content, runID), 250)
  }

  function clearPendingValidation() {
    if (validationTimer) {
      clearTimeout(validationTimer)
      validationTimer = null
    }
  }
}
