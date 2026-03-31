<script lang="ts">
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import EphemeralChatPanel from './ephemeral-chat-panel.svelte'

  let {
    open = $bindable(false),
    organizationId = '',
    projectId = '',
    defaultProviderId = null,
    initialPrompt = '',
  }: {
    open?: boolean
    organizationId?: string
    projectId?: string
    defaultProviderId?: string | null
    initialPrompt?: string
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="sr-only">
      <SheetTitle>Project AI</SheetTitle>
      <SheetDescription>AI assistant for this project.</SheetDescription>
    </SheetHeader>

    {#if open && organizationId && projectId}
      <EphemeralChatPanel
        source="project_sidebar"
        {organizationId}
        {defaultProviderId}
        context={{ projectId }}
        title="Project AI"
        placeholder="Ask anything about this project…"
        {initialPrompt}
      />
    {:else}
      <div class="text-muted-foreground px-6 py-5 text-sm">Project context is unavailable.</div>
    {/if}
  </SheetContent>
</Sheet>
