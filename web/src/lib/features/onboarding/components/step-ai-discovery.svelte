<script lang="ts">
  import { goto } from '$app/navigation'
  import { projectPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Bot, Sparkles, ArrowRight } from '@lucide/svelte'

  let {
    orgId,
    projectId,
    hasWorkflow,
    onOpenProjectAI,
    onComplete,
  }: {
    orgId: string
    projectId: string
    hasWorkflow: boolean
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

  function handleOpenHarnessAI() {
    void goto(projectPath(orgId, projectId, 'workflows'))
    finishOnboarding()
  }
</script>

<div class="space-y-4">
  <div class="bg-muted/50 rounded-lg p-3">
    <p class="text-foreground text-sm font-medium">
      On the final step, clicking any button will finish the tour.
    </p>
    <p class="text-muted-foreground mt-1 text-xs">
      You can try Project AI, open the workflow editor, or click "Got it" to end the tour now.
    </p>
  </div>

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

  <!-- Harness AI -->
  <div class="border-border rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="bg-primary/10 flex size-10 items-center justify-center rounded-lg">
        <Sparkles class="text-primary size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="text-foreground text-sm font-medium">Harness AI</p>
        </div>
        <p class="text-muted-foreground text-xs">Use AI to refine workflow rules and role setup</p>
      </div>
    </div>
    {#if hasWorkflow}
      <Button variant="outline" size="sm" class="mt-3 text-xs" onclick={handleOpenHarnessAI}>
        <ArrowRight class="mr-1 size-3" />
        Open workflow editor
      </Button>
    {:else}
      <p class="text-muted-foreground mt-2 text-xs">Create the agent and workflow first.</p>
    {/if}
  </div>

  <div class="flex justify-end">
    <Button variant="ghost" size="sm" class="text-xs" onclick={finishOnboarding}>Got it</Button>
  </div>
</div>
