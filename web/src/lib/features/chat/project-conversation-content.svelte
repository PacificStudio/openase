<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import ProjectConversationTabStrip from './project-conversation-tab-strip.svelte'
  import ProjectConversationWorkspaceSummary from './project-conversation-workspace-summary.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'
  import type { ProjectConversation } from '$lib/api/chat'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'

  type ProjectConversationTabSummary = {
    id: string
    conversationId: string
    phase: string
    pending: boolean
    hasPendingInterrupt: boolean
    restored: boolean
    draft: string
    entries: ProjectConversationTranscriptEntry[]
  }

  let {
    tabs = [],
    activeTabId = '',
    conversations = [],
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    workspaceDiffError = '',
    entries = [],
    pending = false,
    onSelectTab,
    onCloseTab,
    onConfirmActionProposal,
    onCancelActionProposal,
    onRespondInterrupt,
  }: {
    tabs?: ProjectConversationTabSummary[]
    activeTabId?: string
    conversations?: ProjectConversation[]
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    workspaceDiffError?: string
    entries?: ProjectConversationTranscriptEntry[]
    pending?: boolean
    onSelectTab: (tabId: string) => void
    onCloseTab: (tabId: string) => void
    onConfirmActionProposal: (entryId: string) => void
    onCancelActionProposal: (entryId: string) => void
    onRespondInterrupt: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => void
  } = $props()
</script>

<ProjectConversationTabStrip {tabs} {activeTabId} {conversations} {onSelectTab} {onCloseTab} />

<ProjectConversationWorkspaceSummary
  {conversationId}
  {workspaceDiff}
  loading={workspaceDiffLoading}
  error={workspaceDiffError}
/>

<ScrollArea class="min-h-0 flex-1 px-4 py-4">
  <ProjectConversationTranscript
    {entries}
    {pending}
    {onConfirmActionProposal}
    {onCancelActionProposal}
    {onRespondInterrupt}
  />
</ScrollArea>
