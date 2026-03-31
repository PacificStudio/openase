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
    untrack(() => {
      controller.syncProviders(activeProviders, defaultProviderId)
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
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-4 py-2">
    <div class="flex items-center gap-2">
      <h2 class="text-sm font-semibold">{title}</h2>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        {providerId}
        onProviderChange={(nextProviderId) => void controller.selectProvider(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
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
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || !context.projectId || !providerId || pending}
      >
        <Send class="size-4" />
      </Button>
    </div>
  </div>
</div>
