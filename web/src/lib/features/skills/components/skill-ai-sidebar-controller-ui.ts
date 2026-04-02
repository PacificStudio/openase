export function dismissSkillAISuggestion(setDismissed: (value: boolean) => void) {
  setDismissed(true)
}

export async function handleSkillAIProviderChange(
  providerId: string,
  nextProviderId: string,
  setProviderId: (value: string) => void,
  closeActiveSession: (options: { clearResult: boolean; suppressError?: boolean }) => Promise<void>,
) {
  if (nextProviderId === providerId) return
  setProviderId(nextProviderId)
  await closeActiveSession({ clearResult: true })
}

export function handleSkillAIPromptKeydown(event: KeyboardEvent, handleSend: () => Promise<void>) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  void handleSend()
}
