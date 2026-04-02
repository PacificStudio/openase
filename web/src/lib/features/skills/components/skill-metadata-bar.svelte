<script lang="ts">
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { cn } from '$lib/utils'
  import { Link2, Link2Off } from '@lucide/svelte'

  let {
    skill,
    workflows = [],
    busy = false,
    editDescription = '',
    onEditDescriptionChange,
    onToggleBinding,
  }: {
    skill: Skill
    workflows?: Workflow[]
    busy?: boolean
    editDescription?: string
    onEditDescriptionChange?: (value: string) => void
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => Promise<void> | void
  } = $props()

  function isBound(workflowId: string) {
    return skill.bound_workflows.some((workflow) => workflow.id === workflowId)
  }
</script>

<div class="border-border flex items-center gap-3 border-b px-3 py-1 text-[11px]">
  <!-- Description (inline editable) -->
  <input
    type="text"
    value={editDescription}
    placeholder="Description..."
    class="text-foreground placeholder:text-muted-foreground/50 min-w-0 flex-1 truncate bg-transparent text-[11px] outline-none"
    disabled={busy}
    oninput={(event) => onEditDescriptionChange?.((event.currentTarget as HTMLInputElement).value)}
  />

  <!-- Divider -->
  <div class="bg-border h-3 w-px shrink-0"></div>

  <!-- Workflow bindings -->
  {#if workflows.length > 0}
    <div class="flex shrink-0 items-center gap-1">
      {#each workflows as workflow (workflow.id)}
        {@const bound = isBound(workflow.id)}
        <button
          type="button"
          disabled={busy}
          class={cn(
            'inline-flex items-center gap-0.5 rounded-full border px-1.5 py-px text-[10px] font-medium transition-colors',
            bound
              ? 'border-primary/30 bg-primary/10 text-primary'
              : 'text-muted-foreground/60 hover:text-muted-foreground hover:bg-muted border-transparent',
          )}
          title="{bound ? 'Unbind from' : 'Bind to'} {workflow.name}"
          onclick={() => void onToggleBinding?.(workflow.id, !bound)}
        >
          {#if bound}
            <Link2 class="size-2.5" />
          {:else}
            <Link2Off class="size-2.5" />
          {/if}
          {workflow.name}
        </button>
      {/each}
    </div>
  {/if}

  <!-- Divider -->
  <div class="bg-border h-3 w-px shrink-0"></div>

  <!-- Info -->
  <span class="text-muted-foreground shrink-0 text-[10px]">
    {skill.created_by} · {formatRelativeTime(skill.created_at)}
  </span>
</div>
