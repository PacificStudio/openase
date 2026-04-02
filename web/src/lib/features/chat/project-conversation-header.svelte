<script lang="ts">
  import { Button } from '$ui/button'
  import { Plus, RefreshCcw } from '@lucide/svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'

  let {
    title = 'Project AI',
    providers = [],
    providerId = '',
    providerSelectionDisabled = false,
    resetDisabled = true,
    onProviderChange,
    onCreateTab,
    onResetConversation,
  }: {
    title?: string
    providers?: AgentProvider[]
    providerId?: string
    providerSelectionDisabled?: boolean
    resetDisabled?: boolean
    onProviderChange?: (providerId: string) => void
    onCreateTab?: () => void
    onResetConversation?: () => void
  } = $props()
</script>

<div class="border-border flex items-center gap-2 border-b px-3 py-1.5 pr-12">
  <h2 class="text-xs font-medium">{title}</h2>
  <EphemeralChatProviderSelect
    {providers}
    {providerId}
    disabled={providerSelectionDisabled}
    {onProviderChange}
  />
  <div class="ml-auto flex items-center">
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
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground size-6 p-0"
      aria-label="Reset conversation"
      onclick={onResetConversation}
      disabled={resetDisabled}
    >
      <RefreshCcw class="size-3" />
    </Button>
  </div>
</div>
