<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import ProjectConversationTabStrip from './project-conversation-tab-strip.svelte'
  import ProjectConversationWorkspaceSummary from './project-conversation-workspace-summary.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'
  import type { ProjectConversation } from '$lib/api/chat'
  import type { ProjectConversationTabView } from './project-conversation-panel-labels'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'

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

  const browserOpen = $derived(workspaceBrowserPortal.open)

  // Sync conversation data to the portal so the frame can render the browser
  $effect(() => {
    workspaceBrowserPortal.conversationId = conversationId
    workspaceBrowserPortal.workspaceDiff = workspaceDiff ?? null
    workspaceBrowserPortal.workspaceDiffLoading = workspaceDiffLoading
  })

  // Close the browser when conversation is cleared
  $effect(() => {
    if (!conversationId) {
      workspaceBrowserPortal.close()
    }
  })

  // Close the browser when this component unmounts
  $effect(() => {
    return () => {
      workspaceBrowserPortal.close()
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
    if (conversationId) workspaceBrowserPortal.toggle()
  }}
  onOpenFile={(filePath) => {
    if (conversationId) workspaceBrowserPortal.openToFile(filePath)
  }}
/>

<div class="flex min-h-0 flex-1">
  <div class="min-w-0 flex-1">
    <ScrollArea
      class="h-full px-4 py-4"
      scrollbarYClasses="data-vertical:w-[3px] data-vertical:pr-0"
    >
      <ProjectConversationTranscript {entries} {pending} {onRespondInterrupt} />
    </ScrollArea>
  </div>
</div>
