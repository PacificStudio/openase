<script lang="ts">
  import { untrack } from 'svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import { interruptAgent } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { createProjectConversationController } from './project-conversation-controller.svelte'
  import ProjectConversationComposer from './project-conversation-composer.svelte'
  import ProjectConversationContent from './project-conversation-content.svelte'
  import ProjectConversationHeader from './project-conversation-header.svelte'
  import { runProjectConversationDeleteFlow } from './project-conversation-panel-delete'
  import { interruptFocusedProjectAgent } from './project-conversation-panel-interrupt'
  import {
    describeProjectAIFocus,
    projectAIFocusKey,
    type ProjectAIFocus,
  } from './project-ai-focus'
  import { deriveFocusInterruptTarget } from './project-conversation-panel-focus-action'
  import { getProjectConversationStatusMessage } from './project-conversation-panel-labels'
  import { applyEligibleInitialPrompt } from './project-conversation-panel-prompt'
  import { watchProjectConversationProviders } from './project-conversation-panel-provider-sync'

  let {
    context,
    organizationId = '',
    providers = [],
    defaultProviderId = null,
    focus = null,
    title = 'Project AI',
    placeholder = 'Ask anything about this project…',
    initialPrompt = '',
    onClose,
  }: {
    context: { projectId: string; projectName?: string }
    organizationId?: string
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    focus?: ProjectAIFocus | null
    title?: string
    placeholder?: string
    initialPrompt?: string
    onClose?: () => void
  } = $props()

  let suppressedFocusKey = $state('')
  let loadingProviders = $state(false),
    providerError = $state(''),
    loadedProviders = $state<AgentProvider[]>([])
  let previousRestoreKey = '',
    appliedInitialPromptSignature = $state(''),
    autoDispatchQueueTurnId = $state('')

  const controller = createProjectConversationController({
    getProjectContext: () => ({
      projectId: context.projectId,
      projectName: context.projectName ?? '',
    }),
    getProjectId: () => context.projectId,
    onError: (message) => toastStore.error(message),
  })
  const activeProviders = $derived(providers.length > 0 ? providers : loadedProviders),
    chatProviders = $derived(controller.providers),
    providerId = $derived(controller.providerId),
    conversations = $derived(controller.conversations),
    tabs = $derived(controller.tabs),
    activeTabId = $derived(controller.activeTabId)
  const activeTab = $derived(tabs.find((tab) => tab.id === activeTabId) ?? tabs[0] ?? null)
  const draft = $derived(controller.draft)
  const queuedTurns = $derived(controller.queuedTurns)
  const nextQueuedTurn = $derived(queuedTurns[0] ?? null)
  const entries = $derived(controller.entries)
  const workspaceDiff = $derived(controller.workspaceDiff)
  const workspaceDiffLoading = $derived(controller.workspaceDiffLoading)
  const workspaceDiffError = $derived(controller.workspaceDiffError)
  const pending = $derived(controller.pending)
  const phase = $derived(controller.phase)
  const inputDisabled = $derived(controller.inputDisabled)
  const sendDisabled = $derived(controller.sendDisabled)
  const canQueueTurn = $derived(controller.canQueueTurn)
  const providerSelectionDisabled = $derived(controller.providerSelectionDisabled)
  const statusMessage = $derived(
    getProjectConversationStatusMessage(phase, controller.hasPendingInterrupt),
  )
  const effectiveFocus = $derived(focus && focus.projectId === context.projectId ? focus : null)
  const effectiveFocusKey = $derived(projectAIFocusKey(effectiveFocus))
  const focusForSend = $derived(
    effectiveFocus && suppressedFocusKey !== effectiveFocusKey ? effectiveFocus : null,
  )
  const focusCard = $derived(focusForSend ? describeProjectAIFocus(focusForSend) : null)
  const focusInterruptTarget = $derived(deriveFocusInterruptTarget(focusForSend))
  const showStop = $derived(
    Boolean(activeTab?.conversationId) &&
      !controller.hasPendingInterrupt &&
      (phase === 'connecting_stream' || phase === 'awaiting_reply' || phase === 'stopping_turn'),
  )
  const stopDisabled = $derived(phase === 'stopping_turn')

  $effect(() =>
    watchProjectConversationProviders({
      organizationId,
      hasInlineProviders: providers.length > 0,
      setLoading: (value) => (loadingProviders = value),
      setError: (value) => (providerError = value),
      setProviders: (value) => (loadedProviders = value),
    }),
  )

  $effect(() => {
    const nextProviders = activeProviders
    const nextDefaultProviderId = defaultProviderId
    untrack(() => {
      controller.syncProviders(nextProviders, nextDefaultProviderId)
    })
  })

  $effect(() => {
    const restoreKey = context.projectId
    if (!restoreKey) {
      return
    }
    if (previousRestoreKey) {
      // Project changed but panel already restored - don't re-restore, just persist
      previousRestoreKey = restoreKey
      return
    }
    previousRestoreKey = restoreKey
    appliedInitialPromptSignature = ''

    let cancelled = false
    const restore = async () => {
      await controller.restore()
      if (!cancelled) {
        appliedInitialPromptSignature = applyEligibleInitialPrompt({
          restoreKey,
          nextInitialPrompt: initialPrompt,
          activeTabId: controller.activeTabId,
          appliedInitialPromptSignature,
          activeDraft: controller.draft,
          setDraft: controller.setDraft,
        })
      }
    }

    void restore()
    return () => {
      cancelled = true
    }
  })

  $effect(() => () => controller.dispose())

  $effect(() => {
    if (!effectiveFocusKey) {
      suppressedFocusKey = ''
    }
  })
  $effect(() => {
    const restoreKey = context.projectId
    if (!restoreKey || !controller.activeTabId) {
      return
    }

    const nextInitialPrompt = initialPrompt.trim()
    if (!nextInitialPrompt) {
      return
    }
    appliedInitialPromptSignature = applyEligibleInitialPrompt({
      restoreKey,
      nextInitialPrompt: initialPrompt,
      activeTabId: controller.activeTabId,
      appliedInitialPromptSignature,
      activeDraft: controller.draft,
      setDraft: controller.setDraft,
    })
  })

  $effect(() => {
    const nextQueuedTurnId = nextQueuedTurn?.id ?? ''
    const shouldAutoDispatch =
      !!activeTabId && !!nextQueuedTurnId && phase === 'idle' && !controller.hasPendingInterrupt

    if (!shouldAutoDispatch) {
      autoDispatchQueueTurnId = ''
      return
    }

    if (autoDispatchQueueTurnId === nextQueuedTurnId) {
      return
    }

    autoDispatchQueueTurnId = nextQueuedTurnId
    queueMicrotask(() => {
      if (
        autoDispatchQueueTurnId === nextQueuedTurnId &&
        activeTabId &&
        (controller.queuedTurns[0]?.id ?? '') === nextQueuedTurnId &&
        controller.phase === 'idle' &&
        !controller.hasPendingInterrupt
      ) {
        void controller.sendNextQueuedTurn().finally(() => {
          if (autoDispatchQueueTurnId === nextQueuedTurnId) {
            autoDispatchQueueTurnId = ''
          }
        })
      }
    })
  })

  async function handleSend() {
    const message = controller.draft.trim()
    if (!message) {
      return
    }
    const nextFocus = suppressedFocusKey === effectiveFocusKey ? null : effectiveFocus
    if (controller.queuedTurns.length > 0 || controller.sendDisabled) {
      if (!controller.canQueueTurn || !controller.enqueueTurn(message, nextFocus)) {
        return
      }
      controller.setDraft('')
      suppressedFocusKey = ''
      return
    }

    controller.setDraft('')
    await controller.sendTurn(message, nextFocus)
    suppressedFocusKey = ''
  }

  async function handleInterruptFocusedAgent() {
    await interruptFocusedProjectAgent(
      focusInterruptTarget
        ? {
            agentId: focusInterruptTarget.agentId,
            agentName: focusInterruptTarget.agentName,
            interruptAgent,
            onSuccess: (message) => toastStore.success(message),
            onError: (message) => toastStore.error(message),
          }
        : null,
    )
  }

  async function handleDeleteConversation(conversationId: string, force = false) {
    await runProjectConversationDeleteFlow({
      conversationId,
      force,
      deleteConversation: controller.deleteConversation,
      onDeleted: () => toastStore.success('Project AI conversation deleted.'),
      onError: (message) => toastStore.error(message),
    })
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <ProjectConversationHeader
    {title}
    providers={chatProviders}
    {providerId}
    {providerSelectionDisabled}
    activeTabHasContent={entries.length > 0 || Boolean(activeTab?.conversationId)}
    {conversations}
    openConversationIds={tabs.map((tab) => tab.conversationId).filter(Boolean)}
    onProviderChange={(nextProviderId) => void controller.selectProvider(nextProviderId)}
    onCreateTab={() => controller.createTab()}
    onOpenConversation={(conversationId) => void controller.openConversation(conversationId)}
    onDeleteConversation={(conversationId) => void handleDeleteConversation(conversationId)}
    {onClose}
  />
  <ProjectConversationContent
    {tabs}
    {activeTabId}
    {conversations}
    currentProjectId={context.projectId}
    conversationId={activeTab?.conversationId ?? ''}
    {workspaceDiff}
    {workspaceDiffLoading}
    {workspaceDiffError}
    {entries}
    {pending}
    onSyncWorkspace={() => controller.refreshWorkspaceDiff()}
    onSelectTab={controller.selectTab}
    onCloseTab={controller.closeTab}
    onRespondInterrupt={controller.respondInterrupt}
  />
  <ProjectConversationComposer
    {loadingProviders}
    {providerError}
    providerCount={chatProviders.length}
    statusMessage={statusMessage ?? undefined}
    {focusCard}
    focusActionLabel={focusInterruptTarget ? 'Interrupt Agent' : ''}
    focusActionDisabled={!focusInterruptTarget}
    {queuedTurns}
    hasPendingInterrupt={controller.hasPendingInterrupt}
    {draft}
    {placeholder}
    {inputDisabled}
    {sendDisabled}
    {canQueueTurn}
    {showStop}
    {stopDisabled}
    onFocusAction={handleInterruptFocusedAgent}
    onStop={() => void controller.stopTurn()}
    onDismissFocus={() => (suppressedFocusKey = effectiveFocusKey)}
    onCancelQueuedTurn={(id) => controller.cancelQueuedTurn(id)}
    onDraftChange={controller.setDraft}
    onSend={handleSend}
  />
</div>
