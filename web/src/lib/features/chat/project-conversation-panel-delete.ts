import { ApiError } from '$lib/api/client'
import { chatT } from './i18n'

type DeleteConversationFlowInput = {
  conversationId: string
  force?: boolean
  deleteConversation: (conversationId: string, options?: { force?: boolean }) => Promise<boolean>
  onDeleted: () => void
  onError: (message: string) => void
}

export async function runProjectConversationDeleteFlow(
  input: DeleteConversationFlowInput,
): Promise<void> {
  const force = input.force ?? false
  if (!input.conversationId) {
    return
  }

  if (!force) {
    const confirmed = window.confirm(chatT('chat.confirmDeleteConversation'))
    if (!confirmed) {
      return
    }
  }

  try {
    const deleted = await input.deleteConversation(input.conversationId, { force })
    if (deleted) {
      input.onDeleted()
    }
  } catch (error) {
    if (
      !force &&
      error instanceof ApiError &&
      error.detail.toLowerCase().includes('workspace has uncommitted changes')
    ) {
      const confirmed = window.confirm(
        chatT('chat.confirmDeleteWorkspaceChanges', { errorDetail: error.detail }),
      )
      if (confirmed) {
        await runProjectConversationDeleteFlow({ ...input, force: true })
      }
      return
    }

    input.onError(
      error instanceof ApiError ? error.detail : chatT('chat.deleteConversationFailed'),
    )
  }
}
