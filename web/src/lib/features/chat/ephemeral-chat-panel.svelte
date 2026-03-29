<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { listProviders } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Bot, RefreshCcw, Send } from '@lucide/svelte'
  import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
  import { describeEphemeralChatSessionPolicy } from './session-policy'
  import EphemeralChatTranscript from './ephemeral-chat-transcript.svelte'
  import type { ChatSource } from '$lib/api/chat'

  type EphemeralChatPanelContext = {
    projectId: string
    workflowId?: string
    ticketId?: string
  }

  let {
    source,
    context,
    organizationId = '',
    providers = [],
    defaultProviderId = null,
    title = 'Ask AI',
    description = '',
    placeholder = 'Ask a question about this project.',
    emptyStateTitle = 'Start a conversation',
    emptyStateDescription = 'Use the provider selector to choose an available chat runtime.',
    contextNote = '',
    initialPrompt = '',
    messagePrefix = '',
  }: {
    source: ChatSource
    context: EphemeralChatPanelContext
    organizationId?: string
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    title?: string
    description?: string
    placeholder?: string
    emptyStateTitle?: string
    emptyStateDescription?: string
    contextNote?: string
    initialPrompt?: string
    messagePrefix?: string
  } = $props()

  let prompt = $state('')
  let loadingProviders = $state(false)
  let providerError = $state('')
  let loadedProviders = $state<AgentProvider[]>([])
  let previousContextKey = ''

  const chatController = createEphemeralChatSessionController({
    getSource: () => source,
    onError: (message) => toastStore.error(message),
  })

  const activeProviders = $derived(providers.length > 0 ? providers : loadedProviders)
  const chatProviders = $derived(chatController.providers)
  const providerId = $derived(chatController.providerId)
  const selectedProvider = $derived(chatController.selectedProvider)
  const pending = $derived(chatController.pending)
  const sessionId = $derived(chatController.sessionId)
  const entries = $derived(chatController.entries)
  const providerUnavailable = $derived(
    !loadingProviders && !providerError && chatProviders.length === 0,
  )

  $effect(() => {
    const contextKey = [
      source,
      organizationId,
      context.projectId,
      context.workflowId ?? '',
      context.ticketId ?? '',
      initialPrompt,
      messagePrefix,
    ].join(':')

    if (contextKey === previousContextKey) {
      return
    }

    previousContextKey = contextKey
    prompt = initialPrompt
    void chatController.resetConversation()
  })

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
        if (cancelled) {
          return
        }

        loadedProviders = payload.providers
      } catch (caughtError) {
        if (cancelled) {
          return
        }

        providerError =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load chat providers.'
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
      chatController.syncProviders(nextProviders, nextDefaultProviderId)
    })
  })

  $effect(() => {
    return () => {
      void chatController.dispose()
    }
  })

  function buildOutgoingMessage(message: string) {
    const normalizedPrefix = messagePrefix.trim()
    if (!normalizedPrefix) {
      return message
    }

    return `${normalizedPrefix}\n\nUser request: ${message}`
  }

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !context.projectId || !providerId || pending) {
      return
    }

    prompt = ''

    await chatController.sendTurn({
      message: buildOutgoingMessage(message),
      context,
    })
  }

  async function handleProviderChange(nextProviderId: string) {
    prompt = initialPrompt
    await chatController.selectProvider(nextProviderId)
  }

  async function handleResetConversation() {
    prompt = initialPrompt
    await chatController.resetConversation()
  }

  async function handleConfirmActionProposal(entryId: string) {
    await chatController.confirmActionProposal(entryId)
  }

  function handleCancelActionProposal(entryId: string) {
    chatController.cancelActionProposal(entryId)
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey) {
      return
    }

    event.preventDefault()
    void handleSend()
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <Bot class="text-primary size-4 shrink-0" />
        <h2 class="truncate text-sm font-semibold">{title}</h2>
        {#if selectedProvider}
          <Badge variant="outline" class="text-[10px]">{selectedProvider.name}</Badge>
        {/if}
        {#if sessionId}
          <Badge variant="outline" class="text-[10px]">Context kept</Badge>
        {/if}
      </div>
      {#if description}
        <p class="text-muted-foreground mt-1 text-xs">{description}</p>
      {/if}
    </div>

    <Button
      variant="ghost"
      size="sm"
      onclick={() => void handleResetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <RefreshCcw class="size-4" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <EphemeralChatTranscript
      {entries}
      {pending}
      {emptyStateTitle}
      {emptyStateDescription}
      onConfirmActionProposal={handleConfirmActionProposal}
      onCancelActionProposal={handleCancelActionProposal}
    />
  </ScrollArea>

  <div class="border-border space-y-3 border-t px-4 py-3">
    <EphemeralChatProviderSelect
      providers={chatProviders}
      {providerId}
      onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
    />

    {#if loadingProviders}
      <div class="text-muted-foreground text-xs">Loading available chat providers…</div>
    {:else if providerError}
      <div class="text-destructive text-xs">{providerError}</div>
    {:else if providerUnavailable}
      <div class="text-muted-foreground text-xs">
        No Ephemeral Chat provider is available for this organization.
      </div>
    {/if}

    {#if contextNote}
      <div class="border-border bg-muted/20 rounded-lg border px-3 py-2 text-xs leading-5">
        {contextNote}
      </div>
    {/if}

    <Textarea
      bind:value={prompt}
      rows={4}
      class="text-sm"
      {placeholder}
      disabled={!context.projectId || !providerId || pending}
      onkeydown={handlePromptKeydown}
    />

    <div class="flex items-center justify-between gap-3">
      <p class="text-muted-foreground text-[11px] leading-4">
        {describeEphemeralChatSessionPolicy(source, Boolean(sessionId))}
      </p>
      <Button
        size="sm"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || !context.projectId || !providerId || pending}
      >
        <Send class="size-4" />
        Send
      </Button>
    </div>
  </div>
</div>
