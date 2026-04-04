<script lang="ts">
  import { Button } from '$ui/button'
  import * as Popover from '$ui/popover'
  import { Plus, History, X } from '@lucide/svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { ProjectConversation } from '$lib/api/chat'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
  import ProjectConversationHistoryPopover from './project-conversation-history-popover.svelte'

  let {
    title = 'Project AI',
    providers = [],
    providerId = '',
    providerSelectionDisabled = false,
    activeTabHasContent = false,
    conversations = [],
    openConversationIds = [],
    onProviderChange,
    onCreateTab,
    onOpenConversation,
    onClose,
  }: {
    title?: string
    providers?: AgentProvider[]
    providerId?: string
    providerSelectionDisabled?: boolean
    activeTabHasContent?: boolean
    conversations?: ProjectConversation[]
    openConversationIds?: string[]
    onProviderChange?: (providerId: string) => void
    onCreateTab?: () => void
    onOpenConversation?: (conversationId: string) => void
    onClose?: () => void
  } = $props()

  let historyOpen = $state(false)

  function handleSelectConversation(conversationId: string) {
    historyOpen = false
    onOpenConversation?.(conversationId)
  }
</script>

<div class="border-border flex items-center gap-2 border-b px-3 py-1.5">
  <h2 class="text-xs font-medium">{title}</h2>
  <EphemeralChatProviderSelect
    {providers}
    {providerId}
    disabled={providerSelectionDisabled}
    switchHint={activeTabHasContent ? 'Switching model will open a new session' : ''}
    {onProviderChange}
  />
  <div class="ml-auto flex items-center">
    <Popover.Root bind:open={historyOpen}>
      <Popover.Trigger>
        {#snippet child({ props })}
          <Button
            {...props}
            variant="ghost"
            size="sm"
            class="text-muted-foreground size-6 p-0"
            aria-label="Conversation history"
            disabled={conversations.length === 0}
          >
            <History class="size-3" />
          </Button>
        {/snippet}
      </Popover.Trigger>
      <Popover.Content align="end" sideOffset={6} class="w-72 p-1">
        <div
          class="text-muted-foreground px-2 pt-0.5 pb-0.5 text-[10px] font-medium tracking-wider uppercase"
        >
          History
        </div>
        <ProjectConversationHistoryPopover
          {conversations}
          {openConversationIds}
          onSelect={handleSelectConversation}
        />
      </Popover.Content>
    </Popover.Root>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground h-6 gap-1 px-1.5 text-[11px]"
      aria-label="New Tab"
      onclick={onCreateTab}
      disabled={!providerId}
    >
      <Plus class="size-3" />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground size-6 p-0"
        aria-label="Close panel"
        onclick={onClose}
      >
        <X class="size-3" />
      </Button>
    {/if}
  </div>
</div>
