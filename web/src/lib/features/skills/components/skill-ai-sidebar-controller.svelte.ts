import { untrack } from 'svelte'
import { closeSkillRefinementSession, streamSkillRefinement, type SkillRefinementResultPayload } from '$lib/api/skill-refinement'
import type { AgentProvider, SkillFile } from '$lib/api/contracts'
import { listProviderCapabilityProviders, pickDefaultProviderCapability, shouldKeepProviderCapability } from '$lib/features/chat'
import { buildDiffPreview, type DiffPreview, type SkillSuggestion } from '$lib/features/skills/assistant'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { encodeUTF8Base64 } from './skill-bundle-editor'
import { appendSkillRefinementTranscriptEvent, createSkillRefinementTranscriptState, updateSkillRefinementAnchorState, type SkillRefinementAnchorState } from './skill-refinement-transcript'

type SkillAISidebarInput = {
  getProjectId: () => string | undefined; getProviders: () => AgentProvider[]
  getSkillId: () => string | undefined; getFiles: () => SkillFile[]
  onApplySuggestion?: (files: SkillFile[], focusPath?: string) => void
}

export function createSkillAISidebarController(input: SkillAISidebarInput) {
  let prompt = $state('')
  let refinementProviders = $state<AgentProvider[]>([])
  let providerId = $state('')
  let pending = $state(false)
  let sessionId = $state('')
  let workspacePath = $state('')
  let phase = $state<'' | 'editing' | 'testing' | 'retrying' | 'verified' | 'blocked' | 'unverified'>('')
  let phaseMessage = $state('')
  let attempt = $state(0)
  let result = $state<SkillRefinementResultPayload | null>(null)
  let transcriptState = $state(createSkillRefinementTranscriptState())
  let anchorState = $state<SkillRefinementAnchorState>({})
  let selectedSuggestionPath = $state('')
  let appliedBundleHash = $state('')
  let dismissed = $state(false)
  let previousContextKey = ''
  let abortController: AbortController | null = null

  const transcriptEntries = $derived(transcriptState.entries)
  const textPreviewFiles = $derived(
    result?.candidateFiles.filter((file) => file.encoding === 'utf8' && typeof file.content === 'string') ?? [],
  )
  const suggestion = $derived<SkillSuggestion | null>(
    result?.status === 'verified' && textPreviewFiles.length > 0
      ? {
          summary: result.transcriptSummary || 'Codex verified this draft bundle.',
          files: textPreviewFiles.map((file) => ({
            path: file.path,
            content: file.content ?? '',
          })),
        }
      : null,
  )
  const previewTarget = $derived(
    suggestion?.files.find((file) => file.path === selectedSuggestionPath) ?? suggestion?.files[0] ?? null,
  )
  const preview = $derived<DiffPreview | null>(
    previewTarget
      ? buildDiffPreview(
          input.getFiles().find((file) => file.path === previewTarget.path)?.content ?? '',
          previewTarget.content,
        )
      : null,
  )
  const previewList = $derived(
    suggestion?.files.map((file) => ({
      path: file.path,
      preview: buildDiffPreview(
        input.getFiles().find((current) => current.path === file.path)?.content ?? '',
        file.content,
      ),
    })) ?? [],
  )
  const suggestionAlreadyApplied = $derived(
    (result?.candidateBundleHash && appliedBundleHash === result.candidateBundleHash) ||
      (previewList.length > 0 && previewList.every((item) => !item.preview.hasChanges)),
  )
  const sendDisabled = $derived(
    !input.getProjectId() || !input.getSkillId() || !providerId || !prompt.trim() || pending,
  )

  $effect(() => {
    const nextProviders = input.getProviders()
    const nextDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''
    untrack(() => {
      refinementProviders = listProviderCapabilityProviders(nextProviders, 'skill_ai')
      if (shouldKeepProviderCapability(refinementProviders, providerId, 'skill_ai')) return
      const nextProviderId = pickDefaultProviderCapability(
        refinementProviders,
        nextDefaultProviderId,
        'skill_ai',
      )
      if (providerId && providerId !== nextProviderId) {
        void closeActiveSession({ clearResult: true, suppressError: true })
      }
      providerId = nextProviderId
    })
  })

  $effect(() => {
    const projectId = input.getProjectId()
    const skillId = input.getSkillId()
    const contextKey = projectId && skillId ? `${projectId}:${skillId}` : ''
    if (contextKey === previousContextKey) return
    previousContextKey = contextKey
    prompt = ''
    appliedBundleHash = ''
    dismissed = false
    selectedSuggestionPath = ''
    void closeActiveSession({ clearResult: true, suppressError: true })
  })

  $effect(() => {
    if (!suggestion || suggestion.files.length === 0) {
      selectedSuggestionPath = ''
      return
    }
    if (suggestion.files.some((file) => file.path === selectedSuggestionPath)) return
    selectedSuggestionPath = suggestion.files[0]?.path ?? ''
  })

  $effect(() => {
    return () => {
      void closeActiveSession({ clearResult: false, suppressError: true })
    }
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
    input.onApplySuggestion?.(result.candidateFiles, selectedSuggestionPath || suggestion?.files[0]?.path)
    appliedBundleHash = result.candidateBundleHash ?? ''
  }

  function handleDismiss() { dismissed = true }

  async function handleProviderChange(nextProviderId: string) {
    if (nextProviderId === providerId) return
    providerId = nextProviderId
    await closeActiveSession({ clearResult: true })
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
    event.preventDefault()
    void handleSend()
  }

  return {
    get prompt() { return prompt },
    get refinementProviders() { return refinementProviders },
    get providerId() { return providerId },
    get pending() { return pending },
    get sessionId() { return sessionId },
    get workspacePath() { return workspacePath },
    get phase() { return phase },
    get phaseMessage() { return phaseMessage },
    get attempt() { return attempt },
    get result() { return result },
    get transcriptEntries() { return transcriptEntries },
    get anchorState() { return anchorState },
    get selectedSuggestionPath() { return selectedSuggestionPath },
    get dismissed() { return dismissed },
    get suggestion() { return suggestion },
    get preview() { return preview },
    get suggestionAlreadyApplied() { return suggestionAlreadyApplied },
    get sendDisabled() { return sendDisabled },
    setPrompt(value: string) { prompt = value },
    selectSuggestionPath(path: string) { selectedSuggestionPath = path },
    closeActiveSession,
    handleSend,
    handleApply,
    handleDismiss,
    handleProviderChange,
    handlePromptKeydown,
  }
}
