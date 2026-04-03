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
  import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
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
    placeholder = 'Ask a question…',
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
    placeholder?: string
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
  const pending = $derived(chatController.pending)
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

  let sending = $state(false)
  const sendDisabled = $derived(!context.projectId || !providerId || pending || sending)

  async function handleSend() {
    const message = prompt.trim()
    if (!message || sendDisabled) {
      return
    }

    sending = true
    prompt = ''

    try {
      await chatController.sendTurn({
        message: buildOutgoingMessage(message),
        context,
      })
    } finally {
      sending = false
    }
  }

  async function handleProviderChange(nextProviderId: string) {
    prompt = initialPrompt
    await chatController.selectProvider(nextProviderId)
  }

  async function handleResetConversation() {
    prompt = initialPrompt
    await chatController.resetConversation()
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) {
      return
    }

    event.preventDefault()
    void handleSend()
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-4 py-2">
    <div class="flex items-center gap-2">
      <h2 class="text-sm font-semibold">{title}</h2>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        {providerId}
        onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
      aria-label="Reset conversation"
      onclick={() => void handleResetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <RefreshCcw class="size-3.5" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <EphemeralChatTranscript {entries} {pending} />
  </ScrollArea>

  <div class="border-border border-t px-4 py-3">
    {#if loadingProviders}
      <div class="text-muted-foreground mb-2 text-xs">Loading providers…</div>
    {:else if providerError}
      <div class="text-destructive mb-2 text-xs">{providerError}</div>
    {:else if providerUnavailable}
      <div class="text-muted-foreground mb-2 text-xs">No chat provider available.</div>
    {/if}

    {#if contextNote}
      <div class="text-muted-foreground mb-2 text-[11px] leading-4">{contextNote}</div>
    {/if}

    <div
      class="border-input focus-within:ring-ring flex items-center gap-2 rounded-lg border px-3 py-1.5 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1.5 text-sm shadow-none focus-visible:ring-0"
        {placeholder}
        disabled={!context.projectId || !providerId}
        onkeydown={handlePromptKeydown}
      />
      <Button
        variant="ghost"
        size="sm"
        class="size-7 shrink-0 p-0"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || sendDisabled}
      >
        <Send class="size-4" />
      </Button>
    </div>
  </div>
</div>
