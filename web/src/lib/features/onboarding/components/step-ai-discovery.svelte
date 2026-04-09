<script lang="ts">
  import { Button } from '$ui/button'
  import { Bot, Sparkles } from '@lucide/svelte'

  let {
    onOpenProjectAI,
    onComplete,
  }: {
    onOpenProjectAI: (prompt: string) => void
    onComplete: () => void
  } = $props()

  let completed = $state(false)

  function finishOnboarding() {
    if (completed) return
    completed = true
    onComplete()
  }

  function handleOpenProjectAI(prompt: string) {
    onOpenProjectAI(prompt)
    finishOnboarding()
  }
</script>

<div class="space-y-4">
  <!-- Project AI -->
  <div class="border-border rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="bg-primary/10 flex size-10 items-center justify-center rounded-lg">
        <Bot class="text-primary size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-foreground text-sm font-medium">Project AI</p>
        </div>
        <p class="text-muted-foreground text-xs">
          Use AI to break down requirements and plan follow-up tickets
        </p>
      </div>
    </div>
    <div class="mt-3 flex flex-wrap gap-2">
      <Button
        variant="outline"
        size="sm"
        class="text-xs"
        onclick={() =>
          handleOpenProjectAI(
            'Based on the current project and existing tickets, break down 3 follow-up tickets for me.',
          )}
      >
        <Sparkles class="mr-1 size-3" />
        Break down 3 follow-up tickets
      </Button>
      <Button
        variant="outline"
        size="sm"
        class="text-xs"
        onclick={() => handleOpenProjectAI('What should I do next?')}
      >
        <Sparkles class="mr-1 size-3" />
        What should I do next?
      </Button>
    </div>
  </div>

  <div class="flex justify-end">
    <Button variant="ghost" size="sm" class="text-xs" onclick={finishOnboarding}>Got it</Button>
  </div>
</div>
