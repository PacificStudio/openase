<script lang="ts">
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Input } from '$ui/input'
  import { Clock, Link2, Link2Off } from '@lucide/svelte'

  type SkillHistoryEntry = {
    id: string
    version: number
    created_by: string
    created_at: string
  }

  let {
    skill,
    workflows = [],
    busy = false,
    editDescription = $bindable(''),
    history = [],
    onToggleBinding,
  }: {
    skill: Skill
    workflows?: Workflow[]
    busy?: boolean
    editDescription?: string
    history?: SkillHistoryEntry[]
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => Promise<void> | void
  } = $props()

  function isBound(workflowId: string) {
    return skill.bound_workflows.some((workflow) => workflow.id === workflowId)
  }
</script>

<div class="space-y-4 text-sm" data-testid="skill-metadata-panel">
  <!-- Description -->
  <section class="space-y-1.5">
    <h4 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
      Description
    </h4>
    <Input
      bind:value={editDescription}
      placeholder="Human-readable description"
      class="h-8 text-xs"
      disabled={busy}
    />
  </section>

  <!-- Metadata -->
  <section class="space-y-1">
    <h4 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">Info</h4>
    <div class="text-muted-foreground space-y-0.5 text-xs">
      <p>by {skill.created_by}</p>
      <p>{formatRelativeTime(skill.created_at)}</p>
      <p>{skill.path}</p>
    </div>
  </section>

  <!-- Workflow bindings -->
  <section class="space-y-2">
    <h4 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">Bindings</h4>
    {#if workflows.length > 0}
      <div class="flex flex-wrap gap-1">
        {#each workflows as workflow (workflow.id)}
          {@const bound = isBound(workflow.id)}
          <button
            type="button"
            disabled={busy}
            class="inline-flex h-6 items-center gap-1 rounded-md border px-2 text-[11px] font-medium transition-colors
              {bound
              ? 'border-primary/30 bg-primary/10 text-primary hover:bg-primary/15'
              : 'bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground border-transparent'}"
            title="{bound ? 'Unbind from' : 'Bind to'} {workflow.name}"
            onclick={() => void onToggleBinding?.(workflow.id, !bound)}
          >
            {#if bound}
              <Link2 class="size-2.5" />
            {:else}
              <Link2Off class="size-2.5 opacity-50" />
            {/if}
            {workflow.name}
          </button>
        {/each}
      </div>
    {:else}
      <p class="text-muted-foreground text-xs">No workflows in project.</p>
    {/if}
  </section>

  <!-- Version history -->
  {#if history.length > 0}
    <section class="space-y-2">
      <h4 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
        History
      </h4>
      <div class="space-y-1">
        {#each history as item (item.id)}
          <div class="flex items-center gap-2 text-xs">
            <Clock class="text-muted-foreground size-3 shrink-0" />
            <span class="text-foreground font-medium">v{item.version}</span>
            {#if item.version === skill.current_version}
              <Badge variant="secondary" class="h-4 px-1 text-[9px]">current</Badge>
            {/if}
            <span class="text-muted-foreground truncate">{item.created_by}</span>
            <span class="text-muted-foreground ml-auto shrink-0 text-[10px]">
              {formatRelativeTime(item.created_at)}
            </span>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>
