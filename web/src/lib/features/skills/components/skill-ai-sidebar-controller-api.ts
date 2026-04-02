import type { AgentProvider } from '$lib/api/contracts'
import type { SkillRefinementResultPayload } from '$lib/api/skill-refinement'
import type { DiffPreview, SkillSuggestion } from '$lib/features/skills/assistant'
import type { SkillRefinementAnchorState } from './skill-refinement-transcript'

type SkillAISidebarControllerApiInput = {
  getPrompt: () => string
  getRefinementProviders: () => AgentProvider[]
  getProviderId: () => string
  getPending: () => boolean
  getSessionId: () => string
  getWorkspacePath: () => string
  getPhase: () => '' | 'editing' | 'testing' | 'retrying' | 'verified' | 'blocked' | 'unverified'
  getPhaseMessage: () => string
  getAttempt: () => number
  getResult: () => SkillRefinementResultPayload | null
  getTranscriptEntries: () => ReturnType<
    typeof import('./skill-refinement-transcript').createSkillRefinementTranscriptState
  >['entries']
  getAnchorState: () => SkillRefinementAnchorState
  getSelectedSuggestionPath: () => string
  getDismissed: () => boolean
  getSuggestion: () => SkillSuggestion | null
  getPreview: () => DiffPreview | null
  getSuggestionAlreadyApplied: () => boolean
  getSendDisabled: () => boolean
  setPrompt: (value: string) => void
  selectSuggestionPath: (path: string) => void
  closeActiveSession: (options: { clearResult: boolean; suppressError?: boolean }) => Promise<void>
  handleSend: () => Promise<void>
  handleApply: () => void
  handleDismiss: () => void
  handleProviderChange: (nextProviderId: string) => Promise<void>
  handlePromptKeydown: (event: KeyboardEvent) => void
}

export function createSkillAISidebarControllerApi(input: SkillAISidebarControllerApiInput) {
  return {
    get prompt() {
      return input.getPrompt()
    },
    get refinementProviders() {
      return input.getRefinementProviders()
    },
    get providerId() {
      return input.getProviderId()
    },
    get pending() {
      return input.getPending()
    },
    get sessionId() {
      return input.getSessionId()
    },
    get workspacePath() {
      return input.getWorkspacePath()
    },
    get phase() {
      return input.getPhase()
    },
    get phaseMessage() {
      return input.getPhaseMessage()
    },
    get attempt() {
      return input.getAttempt()
    },
    get result() {
      return input.getResult()
    },
    get transcriptEntries() {
      return input.getTranscriptEntries()
    },
    get anchorState() {
      return input.getAnchorState()
    },
    get selectedSuggestionPath() {
      return input.getSelectedSuggestionPath()
    },
    get dismissed() {
      return input.getDismissed()
    },
    get suggestion() {
      return input.getSuggestion()
    },
    get preview() {
      return input.getPreview()
    },
    get suggestionAlreadyApplied() {
      return input.getSuggestionAlreadyApplied()
    },
    get sendDisabled() {
      return input.getSendDisabled()
    },
    setPrompt: input.setPrompt,
    selectSuggestionPath: input.selectSuggestionPath,
    closeActiveSession: input.closeActiveSession,
    handleSend: input.handleSend,
    handleApply: input.handleApply,
    handleDismiss: input.handleDismiss,
    handleProviderChange: input.handleProviderChange,
    handlePromptKeydown: input.handlePromptKeydown,
  }
}
