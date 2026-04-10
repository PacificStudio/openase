<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import { cn } from '$lib/utils'
  import ProjectConversationTabStrip from './project-conversation-tab-strip.svelte'
  import ProjectConversationWorkspaceBrowser from './project-conversation-workspace-browser.svelte'
  import ProjectConversationWorkspaceSummary from './project-conversation-workspace-summary.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'
  import type { ProjectConversation } from '$lib/api/chat'
  import type { ProjectConversationTabView } from './project-conversation-panel-labels'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'

  let {
    tabs = [],
    activeTabId = '',
    conversations = [],
    currentProjectId = '',
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    workspaceDiffError = '',
    entries = [],
    pending = false,
    onSelectTab,
    onCloseTab,
    onRespondInterrupt,
  }: {
    tabs?: ProjectConversationTabView[]
    activeTabId?: string
    conversations?: ProjectConversation[]
    currentProjectId?: string
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    workspaceDiffError?: string
    entries?: ProjectConversationTranscriptEntry[]
    pending?: boolean
    onSelectTab: (tabId: string) => void
    onCloseTab: (tabId: string) => void
    onRespondInterrupt: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => void
  } = $props()

  let browserOpen = $state(false)

  $effect(() => {
    if (!conversationId) {
      browserOpen = false
    }
  })
</script>

<ProjectConversationTabStrip
  {tabs}
  {activeTabId}
  {conversations}
  {currentProjectId}
  {onSelectTab}
  {onCloseTab}
/>

<ProjectConversationWorkspaceSummary
  {conversationId}
  {workspaceDiff}
  loading={workspaceDiffLoading}
  error={workspaceDiffError}
  {browserOpen}
  onBrowse={() => {
    if (conversationId) browserOpen = !browserOpen
  }}
/>

<div class="flex min-h-0 flex-1">
  <div class="min-w-0 flex-1">
    <ScrollArea
      class={cn('h-full px-4 py-4', browserOpen && 'lg:pr-2')}
      scrollbarYClasses="data-vertical:w-[3px] data-vertical:pr-0"
    >
      <ProjectConversationTranscript {entries} {pending} {onRespondInterrupt} />
    </ScrollArea>
  </div>

  {#if browserOpen}
    <aside class="border-border hidden min-h-0 w-[min(54vw,62rem)] shrink-0 border-l lg:flex">
      <ProjectConversationWorkspaceBrowser
        {conversationId}
        {workspaceDiff}
        {workspaceDiffLoading}
        onClose={() => {
          browserOpen = false
        }}
      />
    </aside>
  {/if}
</div>
