<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import ProjectConversationPanel from './project-conversation-panel.svelte'

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

  const focus = $derived(
    appStore.projectAssistantFocus?.projectId === projectId ? appStore.projectAssistantFocus : null,
  )
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="sr-only">
      <SheetTitle>Project AI</SheetTitle>
      <SheetDescription>AI assistant for this project.</SheetDescription>
    </SheetHeader>

    {#if open && organizationId && projectId}
      <ProjectConversationPanel
        {organizationId}
        {defaultProviderId}
        context={{ projectId }}
        {focus}
        title="Project AI"
        placeholder="Ask anything about this project…"
        {initialPrompt}
      />
    {:else}
      <div class="text-muted-foreground px-6 py-5 text-sm">Project context is unavailable.</div>
    {/if}
  </SheetContent>
</Sheet>
