<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Button } from '$ui/button'
  import * as Select from '$ui/select'
  import { Separator } from '$ui/separator'

  let projectName = $state('OpenASE Demo')
  let description = $state('Autonomous software engineering with AI agents')
  let defaultWorkflow = $state('standard')
  let maxConcurrentAgents = $state('4')

  const workflows = [
    { value: 'standard', label: 'Standard' },
    { value: 'fast-track', label: 'Fast Track' },
    { value: 'review-heavy', label: 'Review Heavy' },
    { value: 'custom', label: 'Custom' },
  ]

  function handleSave() {
    // TODO: persist settings via API
  }
</script>

<div class="max-w-lg space-y-6">
  <div>
    <h2 class="text-base font-semibold text-foreground">General</h2>
    <p class="mt-1 text-sm text-muted-foreground">
      Core project configuration.
    </p>
  </div>

  <Separator />

  <div class="space-y-4">
    <div class="space-y-2">
      <Label for="project-name">Project name</Label>
      <Input id="project-name" bind:value={projectName} />
    </div>

    <div class="space-y-2">
      <Label for="description">Description</Label>
      <Input id="description" bind:value={description} />
    </div>

    <div class="space-y-2">
      <Label>Default workflow</Label>
      <Select.Root
        type="single"
        onValueChange={(v) => { defaultWorkflow = v || 'standard' }}
      >
        <Select.Trigger class="w-full">
          {workflows.find((w) => w.value === defaultWorkflow)?.label ?? 'Select'}
        </Select.Trigger>
        <Select.Content>
          {#each workflows as w (w.value)}
            <Select.Item value={w.value}>{w.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-2">
      <Label for="max-agents">Max concurrent agents</Label>
      <Input
        id="max-agents"
        type="number"
        bind:value={maxConcurrentAgents}
        class="w-24"
      />
      <p class="text-xs text-muted-foreground">
        Limit the number of agents running simultaneously.
      </p>
    </div>
  </div>

  <div class="flex justify-start pt-2">
    <Button onclick={handleSave}>Save changes</Button>
  </div>
</div>
