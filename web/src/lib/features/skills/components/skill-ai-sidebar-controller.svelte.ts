import {
  closeSkillRefinementSession,
  streamSkillRefinement,
  type SkillRefinementResultPayload,
} from '$lib/api/skill-refinement'
import type { AgentProvider } from '$lib/api/contracts'
import type { DiffPreview, SkillSuggestion } from '$lib/features/skills/assistant'
import { toastStore } from '$lib/stores/toast.svelte'
import { encodeUTF8Base64 } from './skill-bundle-editor'
import { createSkillAISidebarControllerApi } from './skill-ai-sidebar-controller-api'
import {
  createSkillAISidebarCleanup,
  syncSkillAIContext,
  syncSkillAIRefinementProviders,
  syncSkillAISuggestionSelection,
} from './skill-ai-sidebar-controller-effects'
import {
  buildSkillAISuggestionPreview,
  buildSkillAISuggestionPreviewList,
  createSkillAISuggestion,
  isSkillAISuggestionAlreadyApplied,
} from './skill-ai-sidebar-controller-preview'
import type { SkillAISidebarInput } from './skill-ai-sidebar-controller-types'
import {
  dismissSkillAISuggestion,
  handleSkillAIProviderChange,
  handleSkillAIPromptKeydown,
} from './skill-ai-sidebar-controller-ui'
import {
  appendSkillRefinementTranscriptEvent,
  createSkillRefinementTranscriptState,
  updateSkillRefinementAnchorState,
  type SkillRefinementAnchorState,
} from './skill-refinement-transcript'

export function createSkillAISidebarController(input: SkillAISidebarInput) {
  let prompt = $state('')
  let refinementProviders = $state<AgentProvider[]>([]),
    providerId = $state(''),
    pending = $state(false),
    sessionId = $state(''),
    workspacePath = $state('')
  let phase = $state<
    '' | 'editing' | 'testing' | 'retrying' | 'verified' | 'blocked' | 'unverified'
  >('')
  let phaseMessage = $state(''),
    attempt = $state(0)
  let result = $state<SkillRefinementResultPayload | null>(null)
  let transcriptState = $state(createSkillRefinementTranscriptState()),
    anchorState = $state<SkillRefinementAnchorState>({})
  let selectedSuggestionPath = $state(''),
    appliedBundleHash = $state('')
  let dismissed = $state(false),
    previousContextKey = ''
  let abortController: AbortController | null = null

  const transcriptEntries = $derived(transcriptState.entries)
  const suggestion = $derived<SkillSuggestion | null>(createSkillAISuggestion(result))
  const preview = $derived<DiffPreview | null>(
    buildSkillAISuggestionPreview(input.getFiles(), suggestion, selectedSuggestionPath),
  )
  const previewList = $derived(buildSkillAISuggestionPreviewList(input.getFiles(), suggestion))
  const suggestionAlreadyApplied = $derived(
    isSkillAISuggestionAlreadyApplied(result, appliedBundleHash, previewList),
  )
  const sendDisabled = $derived(
    !input.getProjectId() || !input.getSkillId() || !providerId || !prompt.trim() || pending,
  )

  $effect(() => {
    syncSkillAIRefinementProviders({
      providers: input.getProviders(),
      providerId,
      setRefinementProviders: (value) => (refinementProviders = value),
      setProviderId: (value) => (providerId = value),
      closeActiveSession,
    })
  })

  $effect(() => {
    syncSkillAIContext({
      projectId: input.getProjectId(),
      skillId: input.getSkillId(),
      previousContextKey,
      setPreviousContextKey: (value) => (previousContextKey = value),
      resetContext: () => {
        prompt = ''
        appliedBundleHash = ''
        dismissed = false
        selectedSuggestionPath = ''
      },
      closeActiveSession,
    })
  })

  $effect(() => {
    syncSkillAISuggestionSelection({
      suggestion,
      selectedSuggestionPath,
      setSelectedSuggestionPath: (value) => (selectedSuggestionPath = value),
    })
  })

  $effect(() => {
    return createSkillAISidebarCleanup(closeActiveSession)
  })

  async function closeActiveSession(options: { clearResult: boolean; suppressError?: boolean }) {
    const activeSessionId = sessionId
    abortController?.abort()
    abortController = null
    pending = false
    sessionId = ''
    workspacePath = ''
    phase = ''
    phaseMessage = ''
    attempt = 0
    transcriptState = createSkillRefinementTranscriptState()
    anchorState = {}
    selectedSuggestionPath = ''
    if (options.clearResult) result = null
    if (!activeSessionId) return

    try {
      await closeSkillRefinementSession(activeSessionId)
    } catch (caughtError) {
      if (options.suppressError) return
      toastStore.error(
        caughtError instanceof Error
          ? caughtError.message
          : 'Failed to close skill refinement session.',
      )
    }
  }

  async function handleSend() {
    const projectId = input.getProjectId()
    const skillId = input.getSkillId()
    if (!projectId || !skillId || sendDisabled) return

    const message = prompt.trim()
    prompt = ''
    pending = true
    result = null
    dismissed = false
    phase = 'editing'
    phaseMessage = 'Preparing the draft bundle for Codex.'
    attempt = 0
    transcriptState = createSkillRefinementTranscriptState()
    anchorState = {}
    appliedBundleHash = ''

    const controller = new AbortController()
    abortController = controller

    try {
      await streamSkillRefinement(
        {
          projectId,
          skillId,
          message,
          providerId,
          files: input.getFiles().map((file) => ({
            path: file.path,
            contentBase64: file.content_base64 ?? encodeUTF8Base64(file.content ?? ''),
            mediaType: file.media_type,
            isExecutable: file.is_executable,
          })),
        },
        {
          signal: controller.signal,
          onEvent: (event) => {
            switch (event.kind) {
              case 'session':
                sessionId = event.payload.sessionId
                workspacePath = event.payload.workspacePath
                return
              case 'status':
                phase = event.payload.phase
                phaseMessage = event.payload.message
                attempt = event.payload.attempt
                transcriptState = appendSkillRefinementTranscriptEvent(transcriptState, event)
                return
              case 'message':
              case 'interrupt_requested':
              case 'thread_status':
              case 'session_state':
              case 'plan_updated':
              case 'diff_updated':
              case 'reasoning_updated':
              case 'thread_compacted':
              case 'session_anchor':
                transcriptState = appendSkillRefinementTranscriptEvent(transcriptState, event)
                anchorState = updateSkillRefinementAnchorState(anchorState, event)
                return
              case 'result':
                result = event.payload
                phase = event.payload.status
                phaseMessage =
                  event.payload.status === 'verified'
                    ? event.payload.transcriptSummary || 'Verification passed.'
                    : event.payload.failureReason || 'Verification did not pass.'
                attempt = event.payload.attempts
                transcriptState = appendSkillRefinementTranscriptEvent(transcriptState, event)
                anchorState = updateSkillRefinementAnchorState(anchorState, event)
                return
              case 'error':
                phase = 'blocked'
                phaseMessage = event.payload.message
                transcriptState = appendSkillRefinementTranscriptEvent(transcriptState, event)
                toastStore.error(event.payload.message)
                return
            }
          },
        },
      )
    } catch (caughtError) {
      if (!(caughtError instanceof DOMException && caughtError.name === 'AbortError')) {
        toastStore.error(
          caughtError instanceof Error ? caughtError.message : 'Skill refinement failed.',
        )
        phase = 'blocked'
        phaseMessage =
          caughtError instanceof Error ? caughtError.message : 'Skill refinement failed.'
      }
    } finally {
      if (abortController === controller) abortController = null
      pending = false
    }
  }

  function handleApply() {
    if (!result || result.status !== 'verified') return
    input.onApplySuggestion?.(
      result.candidateFiles,
      selectedSuggestionPath || suggestion?.files[0]?.path,
    )
    appliedBundleHash = result.candidateBundleHash ?? ''
  }

  async function handleProviderChange(nextProviderId: string) {
    await handleSkillAIProviderChange(
      providerId,
      nextProviderId,
      (value) => (providerId = value),
      closeActiveSession,
    )
  }

  return createSkillAISidebarControllerApi({
    getPrompt: () => prompt,
    getRefinementProviders: () => refinementProviders,
    getProviderId: () => providerId,
    getPending: () => pending,
    getSessionId: () => sessionId,
    getWorkspacePath: () => workspacePath,
    getPhase: () => phase,
    getPhaseMessage: () => phaseMessage,
    getAttempt: () => attempt,
    getResult: () => result,
    getTranscriptEntries: () => transcriptEntries,
    getAnchorState: () => anchorState,
    getSelectedSuggestionPath: () => selectedSuggestionPath,
    getDismissed: () => dismissed,
    getSuggestion: () => suggestion,
    getPreview: () => preview,
    getSuggestionAlreadyApplied: () => suggestionAlreadyApplied,
    getSendDisabled: () => sendDisabled,
    setPrompt: (value) => (prompt = value),
    selectSuggestionPath: (path) => (selectedSuggestionPath = path),
    closeActiveSession,
    handleSend,
    handleApply,
    handleDismiss: () => dismissSkillAISuggestion((value) => (dismissed = value)),
    handleProviderChange,
    handlePromptKeydown: (event) => handleSkillAIPromptKeydown(event, handleSend),
  })
}
