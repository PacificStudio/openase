<script lang="ts">
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import { Clock, Link2, Link2Off, Save, X } from '@lucide/svelte'

  type SkillHistoryEntry = {
    id: string
    version: number
    created_by: string
    created_at: string
  }

  let {
    skill,
    workflows = [],
    loadingDetail = false,
    editing = false,
    busy = false,
    content = '',
    editDescription = $bindable(''),
    editContent = $bindable(''),
    history = [],
    onCancelEdit,
    onSave,
    onToggleBinding,
  }: {
    skill: Skill
    workflows?: Workflow[]
    loadingDetail?: boolean
    editing?: boolean
    busy?: boolean
    content?: string
    editDescription?: string
    editContent?: string
    history?: SkillHistoryEntry[]
    onCancelEdit?: () => void
    onSave?: () => Promise<void> | void
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => Promise<void> | void
  } = $props()

  function isBound(workflowId: string) {
    return skill.bound_workflows.some((workflow) => workflow.id === workflowId)
  }
</script>

{#if loadingDetail}
  <div class="animate-pulse">
    <!-- Skeleton: description + metadata -->
    <div class="border-border space-y-3 border-b px-6 py-4">
      <div class="bg-muted h-4 w-3/4 rounded"></div>
      <div class="flex items-center gap-4">
        <div class="bg-muted h-3 w-16 rounded"></div>
        <div class="bg-muted h-3 w-20 rounded"></div>
      </div>
    </div>
    <!-- Skeleton: workflow bindings -->
    <div class="border-border space-y-2 border-b px-6 py-4">
      <div class="bg-muted h-3 w-28 rounded"></div>
      <div class="flex flex-wrap gap-1.5">
        {#each { length: 3 } as _}
          <div class="bg-muted h-7 w-24 rounded-md"></div>
        {/each}
      </div>
    </div>
    <!-- Skeleton: SKILL.md content -->
    <div class="border-border space-y-2 border-b px-6 py-4">
      <div class="bg-muted h-3 w-16 rounded"></div>
      <div class="bg-muted h-32 w-full rounded-md"></div>
    </div>
    <!-- Skeleton: version history -->
    <div class="space-y-2 px-6 py-4">
      <div class="bg-muted h-3 w-24 rounded"></div>
      {#each { length: 2 } as _}
        <div class="flex items-center gap-3">
          <div class="bg-muted h-3 w-8 rounded"></div>
          <div class="bg-muted h-3 w-20 rounded"></div>
          <div class="bg-muted h-3 w-24 rounded"></div>
        </div>
      {/each}
    </div>
  </div>
{:else}
  <section class="border-border space-y-3 border-b px-6 py-4">
    {#if editing}
      <div class="space-y-1.5">
        <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
          Description
        </span>
        <Input
          bind:value={editDescription}
          placeholder="Human-readable description"
          class="h-8 text-sm"
          disabled={busy}
        />
      </div>
    {:else if skill.description}
      <p class="text-muted-foreground text-sm leading-relaxed">{skill.description}</p>
    {/if}

    <div class="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
      <span>by {skill.created_by}</span>
      <span>{formatRelativeTime(skill.created_at)}</span>
      {#if !skill.is_enabled}
        <Badge variant="secondary" class="text-muted-foreground px-1.5 py-0.5 text-[10px]">
          disabled
        </Badge>
      {/if}
    </div>
  </section>

  <section class="border-border space-y-2 border-b px-6 py-4">
    <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
      Workflow bindings
    </h3>
    {#if workflows.length > 0}
      <div class="flex flex-wrap gap-1.5">
        {#each workflows as workflow (workflow.id)}
          {@const bound = isBound(workflow.id)}
          <button
            type="button"
            disabled={busy}
            class="inline-flex h-7 items-center gap-1.5 rounded-md border px-2.5 text-xs font-medium transition-colors
              {bound
              ? 'border-primary/30 bg-primary/10 text-primary hover:bg-primary/15'
              : 'bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground border-transparent'}"
            title="{bound ? 'Unbind from' : 'Bind to'} {workflow.name}"
            onclick={() => void onToggleBinding?.(workflow.id, !bound)}
          >
            {#if bound}
              <Link2 class="size-3" />
            {:else}
              <Link2Off class="size-3 opacity-50" />
            {/if}
            {workflow.name}
          </button>
        {/each}
      </div>
    {:else}
      <p class="text-muted-foreground text-xs">No workflows in this project.</p>
    {/if}
  </section>

  <section class="border-border space-y-2 border-b px-6 py-4">
    <div class="flex items-center justify-between">
      <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
        SKILL.md
      </h3>
      {#if editing}
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            class="h-7 gap-1 px-2 text-xs"
            onclick={onCancelEdit}
            disabled={busy}
          >
            <X class="size-3" />
            Cancel
          </Button>
          <Button
            size="sm"
            class="h-7 gap-1 px-2 text-xs"
            onclick={() => void onSave?.()}
            disabled={busy}
          >
            <Save class="size-3" />
            {busy ? 'Saving…' : 'Publish'}
          </Button>
        </div>
      {/if}
    </div>
    {#if editing}
      <Textarea bind:value={editContent} class="min-h-64 font-mono text-sm" disabled={busy} />
    {:else}
      <pre
        class="bg-muted/40 max-h-96 overflow-auto rounded-lg border p-4 text-sm leading-relaxed whitespace-pre-wrap">{content ||
          '(empty)'}</pre>
    {/if}
  </section>

  {#if history.length > 0}
    <section class="space-y-2 px-6 py-4">
      <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
        Version history
      </h3>
      <div class="space-y-1.5">
        {#each history as item (item.id)}
          <div class="flex items-center gap-3 text-xs">
            <Clock class="text-muted-foreground size-3 shrink-0" />
            <span class="text-foreground font-medium">v{item.version}</span>
            {#if item.version === skill.current_version}
              <Badge variant="secondary" class="h-4 px-1.5 text-[10px]">current</Badge>
            {/if}
            <span class="text-muted-foreground">{item.created_by}</span>
            <span class="text-muted-foreground ml-auto shrink-0">
              {formatRelativeTime(item.created_at)}
            </span>
          </div>
        {/each}
      </div>
    </section>
  {/if}
{/if}
