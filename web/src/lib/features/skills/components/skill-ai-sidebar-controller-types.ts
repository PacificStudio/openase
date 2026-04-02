import type { AgentProvider, SkillFile } from '$lib/api/contracts'

export type SkillAISidebarInput = {
  getProjectId: () => string | undefined
  getProviders: () => AgentProvider[]
  getSkillId: () => string | undefined
  getFiles: () => SkillFile[]
  onApplySuggestion?: (files: SkillFile[], focusPath?: string) => void
}
