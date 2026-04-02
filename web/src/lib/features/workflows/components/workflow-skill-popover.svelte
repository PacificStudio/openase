<script lang="ts">
  import { cn } from '$lib/utils'
  import { getSkill } from '$lib/api/openase'
  import { Skeleton } from '$ui/skeleton'
  import { Button } from '$ui/button'
  import * as Popover from '$ui/popover'
  import { ChevronDown, ChevronUp, Link, Unlink } from '@lucide/svelte'
  import type { SkillState } from '../model'

  let {
    skill,
    onToggle,
  }: {
    skill: SkillState
    onToggle?: (skill: SkillState) => void
  } = $props()

  let open = $state(false)
  let content = $state<string | null>(null)
  let loadingContent = $state(false)
  let loadError = $state('')
  let expanded = $state(false)

  async function fetchContent() {
    if (content !== null || loadingContent) return
    loadingContent = true
    loadError = ''
    try {
      const payload = await getSkill(skill.id)
      content = payload.content || ''
    } catch {
      loadError = 'Failed to load skill content.'
    } finally {
      loadingContent = false
    }
  }

  function handleOpenChange(next: boolean) {
    open = next
    if (next) {
      void fetchContent()
    }
  }

  function handleToggle() {
    onToggle?.(skill)
    open = false
  }

  const contentLines = $derived(content?.split('\n') ?? [])
  const previewLines = $derived(expanded ? contentLines : contentLines.slice(0, 12))
  const hasMore = $derived(contentLines.length > 12)
</script>

<Popover.Root onOpenChange={handleOpenChange} bind:open>
  <Popover.Trigger>
    <button
      type="button"
      class={cn(
        'flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] transition-colors',
        skill.bound
          ? 'border-primary/40 bg-primary/10 text-foreground'
          : 'border-border text-muted-foreground hover:bg-muted',
      )}
    >
      <span
        class={cn(
          'size-1.5 rounded-full',
          skill.bound ? 'animate-pulse-dot bg-primary' : 'bg-muted-foreground/40',
        )}
      ></span>
      {skill.name}
    </button>
  </Popover.Trigger>

  <Popover.Content class="w-96 p-0" align="start" sideOffset={6}>
    <!-- Header -->
    <div class="border-border flex items-center justify-between border-b px-3 py-2.5">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span
            class={cn(
              'size-2 shrink-0 rounded-full',
              skill.bound ? 'bg-primary' : 'bg-muted-foreground/30',
            )}
          ></span>
          <span class="text-foreground truncate text-sm font-medium">{skill.name}</span>
          <span
            class={cn(
              'shrink-0 rounded-full px-1.5 py-px text-[10px] font-medium',
              skill.bound ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground',
            )}
          >
            {skill.bound ? 'Bound' : 'Unbound'}
          </span>
        </div>
        {#if skill.description}
          <p class="text-muted-foreground mt-1 text-xs leading-relaxed">{skill.description}</p>
        {/if}
      </div>
    </div>

    <!-- Content preview -->
    <div class="border-border border-b">
      <div class="px-3 py-2">
        <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
          SKILL.md
        </span>
      </div>
      <div class="max-h-64 overflow-y-auto px-3 pb-2">
        {#if loadingContent}
          <div class="space-y-1.5 pb-1">
            <Skeleton class="h-3 w-[60%]" />
            <Skeleton class="h-3 w-[85%]" />
            <Skeleton class="h-3 w-[45%]" />
            <Skeleton class="h-3 w-[70%]" />
            <Skeleton class="h-3 w-[55%]" />
          </div>
        {:else if loadError}
          <p class="text-destructive pb-1 text-xs">{loadError}</p>
        {:else if content !== null}
          <pre
            class="text-foreground/80 overflow-x-auto font-mono text-[11px] leading-relaxed break-words whitespace-pre-wrap">{previewLines.join(
              '\n',
            )}</pre>
          {#if hasMore}
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground mt-1 flex items-center gap-1 text-[11px] transition-colors"
              onclick={() => (expanded = !expanded)}
            >
              {#if expanded}
                <ChevronUp class="size-3" />
                Show less
              {:else}
                <ChevronDown class="size-3" />
                {contentLines.length - 12} more lines
              {/if}
            </button>
          {/if}
        {:else}
          <p class="text-muted-foreground pb-1 text-xs italic">No content.</p>
        {/if}
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center justify-between px-3 py-2">
      <span class="text-muted-foreground truncate font-mono text-[10px]">{skill.path}</span>
      <Button
        size="sm"
        variant={skill.bound ? 'outline' : 'default'}
        class="h-7 gap-1.5 text-xs"
        onclick={handleToggle}
      >
        {#if skill.bound}
          <Unlink class="size-3" />
          Unbind
        {:else}
          <Link class="size-3" />
          Bind
        {/if}
      </Button>
    </div>
  </Popover.Content>
</Popover.Root>
