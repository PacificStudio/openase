import { untrack } from 'svelte'
import type { AgentProvider } from '$lib/api/contracts'
import {
  listProviderCapabilityProviders,
  pickDefaultProviderCapability,
  shouldKeepProviderCapability,
} from '$lib/features/chat'
import { appStore } from '$lib/stores/app.svelte'
import type { SkillSuggestion } from '$lib/features/skills/assistant'

type SkillAIProviderEffectInput = {
  providers: AgentProvider[]
  providerId: string
  setRefinementProviders: (value: AgentProvider[]) => void
  setProviderId: (value: string) => void
  closeActiveSession: (options: { clearResult: boolean; suppressError?: boolean }) => Promise<void>
}

type SkillAIContextEffectInput = {
  projectId: string | undefined
  skillId: string | undefined
  previousContextKey: string
  setPreviousContextKey: (value: string) => void
  resetContext: () => void
  closeActiveSession: (options: { clearResult: boolean; suppressError?: boolean }) => Promise<void>
}

type SkillAISuggestionEffectInput = {
  suggestion: SkillSuggestion | null
  selectedSuggestionPath: string
  setSelectedSuggestionPath: (value: string) => void
}

export function syncSkillAIRefinementProviders(input: SkillAIProviderEffectInput) {
  const nextDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''
  untrack(() => {
    const refinementProviders = listProviderCapabilityProviders(input.providers, 'skill_ai')
    input.setRefinementProviders(refinementProviders)
    if (shouldKeepProviderCapability(refinementProviders, input.providerId, 'skill_ai')) return

    const nextProviderId = pickDefaultProviderCapability(
      refinementProviders,
      nextDefaultProviderId,
      'skill_ai',
    )
    if (input.providerId && input.providerId !== nextProviderId) {
      void input.closeActiveSession({ clearResult: true, suppressError: true })
    }
    input.setProviderId(nextProviderId)
  })
}

export function syncSkillAIContext(input: SkillAIContextEffectInput) {
  const contextKey = input.projectId && input.skillId ? `${input.projectId}:${input.skillId}` : ''
  if (contextKey === input.previousContextKey) return
  input.setPreviousContextKey(contextKey)
  input.resetContext()
  void input.closeActiveSession({ clearResult: true, suppressError: true })
}

export function syncSkillAISuggestionSelection(input: SkillAISuggestionEffectInput) {
  if (!input.suggestion || input.suggestion.files.length === 0) {
    input.setSelectedSuggestionPath('')
    return
  }
  if (input.suggestion.files.some((file) => file.path === input.selectedSuggestionPath)) return
  input.setSelectedSuggestionPath(input.suggestion.files[0]?.path ?? '')
}

export function createSkillAISidebarCleanup(
  closeActiveSession: (options: { clearResult: boolean; suppressError?: boolean }) => Promise<void>,
) {
  return () => {
    void closeActiveSession({ clearResult: false, suppressError: true })
  }
}
