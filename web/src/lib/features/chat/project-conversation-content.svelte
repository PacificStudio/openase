<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import ProjectConversationTabStrip from './project-conversation-tab-strip.svelte'
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
/>

<ScrollArea class="min-h-0 flex-1 px-4 py-4">
  <ProjectConversationTranscript {entries} {pending} {onRespondInterrupt} />
</ScrollArea>
