import { projectConversationHasPendingInterrupt } from './project-conversation-controller-helpers'
import type { ProjectAIFocus } from './project-ai-focus'
import type { ProjectConversationTabState } from './project-conversation-controller-state'
import { sendNextQueuedProjectConversationTurn } from './project-conversation-controller-actions'

export function createProjectConversationQueuedTurnDispatcher(input: {
  getTabs: () => ProjectConversationTabState[]
  sendTurnInTab: (
    tab: ProjectConversationTabState,
    message: string,
    focus: ProjectAIFocus | null,
  ) => Promise<boolean>
  touch: () => void
}) {
  let queuedTurnDispatchScheduled = false
  const autoDispatchQueuedTurnIDByTab = new Map<string, string>()

  function schedule() {
    if (queuedTurnDispatchScheduled) {
      return
    }

    queuedTurnDispatchScheduled = true
    queueMicrotask(() => {
      queuedTurnDispatchScheduled = false

      for (const tab of input.getTabs()) {
        const nextQueuedTurnId = tab.queuedTurns[0]?.id ?? ''
        const shouldAutoDispatch =
          !!nextQueuedTurnId &&
          !!tab.projectId &&
          !!tab.providerId &&
          tab.phase === 'idle' &&
          !projectConversationHasPendingInterrupt(tab.entries)

        if (!shouldAutoDispatch) {
          autoDispatchQueuedTurnIDByTab.delete(tab.id)
          continue
        }

        if (autoDispatchQueuedTurnIDByTab.get(tab.id) === nextQueuedTurnId) {
          continue
        }

        autoDispatchQueuedTurnIDByTab.set(tab.id, nextQueuedTurnId)
        queueMicrotask(() => {
          const liveTab = input.getTabs().find((item) => item.id === tab.id) ?? null
          if (
            !liveTab ||
            liveTab.phase !== 'idle' ||
            projectConversationHasPendingInterrupt(liveTab.entries) ||
            (liveTab.queuedTurns[0]?.id ?? '') !== nextQueuedTurnId
          ) {
            if (autoDispatchQueuedTurnIDByTab.get(tab.id) === nextQueuedTurnId) {
              autoDispatchQueuedTurnIDByTab.delete(tab.id)
            }
            return
          }

          void sendNextQueuedProjectConversationTurn({
            tab: liveTab,
            sendTurnInTab: input.sendTurnInTab,
          }).then((sent) => {
            if (autoDispatchQueuedTurnIDByTab.get(tab.id) === nextQueuedTurnId) {
              autoDispatchQueuedTurnIDByTab.delete(tab.id)
            }
            if (sent) {
              input.touch()
            }
          })
        })
      }
    })
  }

  return { schedule }
}
