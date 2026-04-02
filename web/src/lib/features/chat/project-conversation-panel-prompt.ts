type EligibleInitialPromptInput = {
  restoreKey: string
  nextInitialPrompt: string
  activeTabId: string
  appliedInitialPromptSignature: string
  activeDraft: string
}

type EligibleInitialPromptResult = {
  signature: string
  shouldApplyDraft: boolean
}

export function getEligibleInitialPromptSignature({
  restoreKey,
  nextInitialPrompt,
  activeTabId,
  appliedInitialPromptSignature,
  activeDraft,
}: EligibleInitialPromptInput): EligibleInitialPromptResult | null {
  const trimmedPrompt = nextInitialPrompt.trim()
  if (!trimmedPrompt || !activeTabId) {
    return null
  }

  const signature = `${restoreKey}:${nextInitialPrompt}`
  if (signature === appliedInitialPromptSignature) {
    return null
  }

  return {
    signature,
    shouldApplyDraft: activeDraft.trim().length === 0,
  }
}
