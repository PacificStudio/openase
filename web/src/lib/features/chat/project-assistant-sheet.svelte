<script lang="ts">
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import EphemeralChatPanel from './ephemeral-chat-panel.svelte'

  let {
    open = $bindable(false),
    organizationId = '',
    projectId = '',
    projectName = '',
    defaultProviderId = null,
    initialPrompt = '',
  }: {
    open?: boolean
    organizationId?: string
    projectId?: string
    projectName?: string
    defaultProviderId?: string | null
    initialPrompt?: string
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <SheetTitle>Ask AI</SheetTitle>
      <SheetDescription>
        {projectName
          ? `Ask AI about ${projectName}, recent activity, blocked work, or platform setup.`
          : 'Ask AI about this project.'}
      </SheetDescription>
    </SheetHeader>

    {#if open && organizationId && projectId}
      <EphemeralChatPanel
        source="project_sidebar"
        {organizationId}
        {defaultProviderId}
        context={{ projectId }}
        title="Project AI"
        description={projectName
          ? `Project context for ${projectName}.`
          : 'Project context loaded.'}
        placeholder="Ask about project health, blocked work, staffing, or setup."
        emptyStateTitle="Project context is ready"
        emptyStateDescription="Ask about ticket flow, recent activity, staffing needs, or platform setup for this project."
        {initialPrompt}
      />
    {:else}
      <div class="text-muted-foreground px-6 py-5 text-sm">Project context is unavailable.</div>
    {/if}
  </SheetContent>
</Sheet>
