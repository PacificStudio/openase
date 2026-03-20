<script lang="ts">
  import { Cable } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import { workflowHasSkill } from '$lib/features/workspace/mappers'
  import type { Skill, Workflow } from '$lib/features/workspace/types'

  let {
    skills = [],
    selectedWorkflow = null,
    busy = false,
    pendingSkillName = '',
    harnessDirty = false,
    onToggleSkill,
  }: {
    skills?: Skill[]
    selectedWorkflow?: Workflow | null
    busy?: boolean
    pendingSkillName?: string
    harnessDirty?: boolean
    onToggleSkill?: (skill: Skill) => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/70">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Cable class="size-4" />
      <span>Skills</span>
    </CardTitle>
    <CardDescription>Bind project skills onto the selected workflow.</CardDescription>
  </CardHeader>

  <CardContent class="space-y-3">
    {#if skills.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        No skills discovered in this project.
      </div>
    {:else}
      {#each skills as skill}
        <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-sm font-semibold">{skill.name}</p>
                <Badge variant={skill.is_builtin ? 'secondary' : 'outline'}>
                  {skill.is_builtin ? 'built-in' : 'project'}
                </Badge>
              </div>
              <p class="text-muted-foreground mt-2 text-sm">
                {skill.description || 'No description yet.'}
              </p>
            </div>
            {#if selectedWorkflow}
              <Button
                type="button"
                size="sm"
                variant={workflowHasSkill(skill, selectedWorkflow.id) ? 'outline' : 'default'}
                disabled={busy || harnessDirty}
                onclick={() => onToggleSkill?.(skill)}
              >
                {workflowHasSkill(skill, selectedWorkflow.id) ? 'Unbind' : 'Bind'}
                {#if busy && pendingSkillName === skill.name}
                  {' '}…
                {/if}
              </Button>
            {/if}
          </div>
        </div>
      {/each}
    {/if}

    {#if !selectedWorkflow}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-2xl border border-dashed px-4 py-4 text-sm"
      >
        Select a workflow to manage skills.
      </div>
    {:else if harnessDirty}
      <div
        class="rounded-2xl border border-amber-500/25 bg-amber-500/10 px-4 py-4 text-sm text-amber-950"
      >
        Save the harness draft before changing skill bindings.
      </div>
    {/if}
  </CardContent>
</Card>
