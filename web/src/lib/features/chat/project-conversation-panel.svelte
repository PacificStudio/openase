<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { listProviders } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { RefreshCcw, Send } from '@lucide/svelte'
  import { createProjectConversationController } from './project-conversation-controller.svelte'
  import type { ProjectConversationPhase } from './project-conversation-controller-helpers'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'

  let {
    context,
    organizationId = '',
    providers = [],
    defaultProviderId = null,
    title = 'Project AI',
    placeholder = 'Ask anything about this project…',
    initialPrompt = '',
  }: {
    context: { projectId: string }
    organizationId?: string
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    title?: string
    placeholder?: string
    initialPrompt?: string
  } = $props()

  let prompt = $state('')
  let loadingProviders = $state(false)
  let providerError = $state('')
  let loadedProviders = $state<AgentProvider[]>([])
  let previousRestoreKey = ''

  const controller = createProjectConversationController({
    getProjectId: () => context.projectId,
    onError: (message) => toastStore.error(message),
  })

  const activeProviders = $derived(providers.length > 0 ? providers : loadedProviders)
  const chatProviders = $derived(controller.providers)
  const providerId = $derived(controller.providerId)
  const entries = $derived(controller.entries)
  const pending = $derived(controller.pending)
  const phase = $derived(controller.phase)
  const inputDisabled = $derived(controller.inputDisabled)
  const providerSelectionDisabled = $derived(controller.providerSelectionDisabled)
  const statusMessage = $derived(getStatusMessage(phase, controller.hasPendingInterrupt))

  $effect(() => {
    if (providers.length > 0 || !organizationId) {
      loadingProviders = false
      providerError = ''
      loadedProviders = []
      return
    }

    let cancelled = false

    const load = async () => {
      loadingProviders = true
      providerError = ''

      try {
        const payload = await listProviders(organizationId)
        if (!cancelled) {
          loadedProviders = payload.providers
        }
      } catch (caughtError) {
        if (!cancelled) {
          providerError =
            caughtError instanceof ApiError ? caughtError.detail : 'Failed to load chat providers.'
        }
      } finally {
        if (!cancelled) {
          loadingProviders = false
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const nextProviders = activeProviders
    const nextDefaultProviderId = defaultProviderId
    untrack(() => {
      controller.syncProviders(nextProviders, nextDefaultProviderId)
    })
  })

  $effect(() => {
    const restoreKey = `${context.projectId}:${providerId}`
    if (!context.projectId || !providerId || restoreKey === previousRestoreKey) {
      return
    }
    previousRestoreKey = restoreKey
    prompt = initialPrompt
    void controller.restore()
  })

  $effect(() => {
    return () => {
      controller.dispose()
    }
  })

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !context.projectId || !providerId || pending) {
      return
    }

    prompt = ''
    await controller.sendTurn(message)
  }

  function getStatusMessage(
    currentPhase: ProjectConversationPhase,
    hasPendingInterrupt: boolean,
  ): string | null {
    if (hasPendingInterrupt) {
      return 'Additional input is required before the conversation can continue.'
    }

    switch (currentPhase) {
      case 'restoring':
        return 'Restoring the latest project conversation…'
      case 'creating_conversation':
        return 'Creating a fresh project conversation…'
      case 'connecting_stream':
        return 'Connecting the live conversation stream…'
      case 'submitting_turn':
        return 'Sending your message…'
      case 'awaiting_reply':
        return 'Waiting for the assistant reply…'
      case 'resetting':
        return 'Resetting the current conversation…'
      default:
        return null
    }
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-4 py-2">
    <div class="flex items-center gap-2">
      <h2 class="text-sm font-semibold">{title}</h2>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        {providerId}
        disabled={providerSelectionDisabled}
        onProviderChange={(nextProviderId) => void controller.selectProvider(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
      aria-label="Reset conversation"
      onclick={() => void controller.resetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <RefreshCcw class="size-3.5" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <ProjectConversationTranscript
      {entries}
      {pending}
      onConfirmActionProposal={(entryId) => controller.confirmActionProposal(entryId)}
      onRespondInterrupt={(payload) => controller.respondInterrupt(payload)}
    />
  </ScrollArea>

  <div class="border-border border-t px-4 py-3">
    {#if loadingProviders}
      <div class="text-muted-foreground mb-2 text-xs">Loading providers…</div>
    {:else if providerError}
      <div class="text-destructive mb-2 text-xs">{providerError}</div>
    {:else if chatProviders.length === 0}
      <div class="text-muted-foreground mb-2 text-xs">No chat provider available.</div>
    {:else if statusMessage}
      <div class="text-muted-foreground mb-2 text-xs">{statusMessage}</div>
    {/if}

    <div
      class="border-input focus-within:ring-ring flex items-center gap-2 rounded-lg border px-3 py-1.5 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1.5 text-sm shadow-none focus-visible:ring-0"
        {placeholder}
        disabled={inputDisabled}
        onkeydown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
            event.preventDefault()
            void handleSend()
          }
        }}
      />
      <Button
        variant="ghost"
        size="sm"
        class="size-7 shrink-0 p-0"
        aria-label="Send message"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || inputDisabled}
      >
        <Send class="size-4" />
      </Button>
    </div>
  </div>
</div>
