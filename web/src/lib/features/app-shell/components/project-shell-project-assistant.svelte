<script lang="ts">
  import { ProjectConversationPanel, type ProjectAIFocus } from '$lib/features/chat'
  import { viewport } from '$lib/stores/viewport.svelte'
  import { cn } from '$lib/utils'
  import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '$ui/sheet'

  let {
    organizationId,
    projectId,
    projectName = '',
    defaultProviderId,
    focus = null,
    open = false,
    initialPrompt = '',
    width = $bindable(380),
    resizing = $bindable(false),
    onClose,
  }: {
    organizationId: string
    projectId: string
    projectName?: string
    defaultProviderId: string | null
    focus?: ProjectAIFocus | null
    open?: boolean
    initialPrompt?: string
    width?: number
    resizing?: boolean
    onClose: () => void
  } = $props()

  const MIN_ASSISTANT_WIDTH = 280
  const MAX_ASSISTANT_WIDTH = 640
  const isMobile = $derived(viewport.isMobile)

  function handleResizeStart(event: PointerEvent) {
    event.preventDefault()
    resizing = true
    const startX = event.clientX
    const startWidth = width

    function onMove(moveEvent: PointerEvent) {
      const delta = startX - moveEvent.clientX
      width = Math.min(MAX_ASSISTANT_WIDTH, Math.max(MIN_ASSISTANT_WIDTH, startWidth + delta))
    }

    function onUp() {
      resizing = false
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }
</script>

{#if isMobile}
  <Sheet
    {open}
    onOpenChange={(nextOpen) => {
      if (!nextOpen) onClose()
    }}
  >
    <SheetContent
      side="bottom"
      class="flex h-[calc(100dvh-48px)] flex-col gap-0 p-0"
      showCloseButton={false}
    >
      <SheetHeader class="sr-only">
        <SheetTitle>Project AI</SheetTitle>
        <SheetDescription>AI assistant for the current project</SheetDescription>
      </SheetHeader>
      <div class="flex min-h-0 flex-1 flex-col">
        <ProjectConversationPanel
          {organizationId}
          {defaultProviderId}
          context={{ projectId, projectName }}
          {focus}
          title="Project AI"
          placeholder="Ask anything about this project…"
          {initialPrompt}
          {onClose}
        />
      </div>
    </SheetContent>
  </Sheet>
{:else if open}
  <aside class="bg-background relative flex h-full shrink-0 flex-col" style="width: {width}px">
    <div
      class={cn(
        'absolute inset-y-0 left-0 z-20 w-1 cursor-col-resize transition-colors',
        resizing ? 'bg-primary' : 'bg-border hover:bg-primary/50',
      )}
      role="separator"
      aria-orientation="vertical"
      onpointerdown={handleResizeStart}
    ></div>
    <div class="flex h-full min-w-0 flex-col pl-1">
      <ProjectConversationPanel
        {organizationId}
        {defaultProviderId}
        context={{ projectId, projectName }}
        {focus}
        title="Project AI"
        placeholder="Ask anything about this project…"
        {initialPrompt}
        {onClose}
      />
    </div>
  </aside>
{/if}
