<script lang="ts">
  import { cn } from '$lib/utils'
  import { getSkill } from '$lib/api/openase'
  import { Button } from '$ui/button'
  import { Skeleton } from '$ui/skeleton'
  import * as Popover from '$ui/popover'
  import { Blocks, Link, Settings, Unlink } from '@lucide/svelte'
  import type { SkillState } from '../model'

  let {
    skillStates,
    skillsSettingsHref,
    onToggleSkill,
  }: {
    skillStates: SkillState[]
    skillsSettingsHref: string | null
    onToggleSkill?: (skill: SkillState) => void
  } = $props()

  let open = $state(false)
  let expandedSkillId = $state<string | null>(null)
  let contentCache = $state<Record<string, string>>({})
  let loadingSkillId = $state<string | null>(null)
  let loadError = $state('')

  const boundCount = $derived(skillStates.filter((s) => s.bound).length)

  function handlePopoverChange(next: boolean) {
    open = next
    if (!next) {
      expandedSkillId = null
      loadError = ''
    }
  }

  async function toggleExpand(skill: SkillState) {
    if (expandedSkillId === skill.id) {
      expandedSkillId = null
      return
    }
    expandedSkillId = skill.id

    if (contentCache[skill.id] !== undefined) return

    loadingSkillId = skill.id
    loadError = ''
    try {
      const payload = await getSkill(skill.id)
      contentCache = { ...contentCache, [skill.id]: payload.content || '' }
    } catch {
      loadError = 'Failed to load skill content.'
    } finally {
      loadingSkillId = null
    }
  }
</script>

<Popover.Root bind:open onOpenChange={handlePopoverChange}>
  <Popover.Trigger>
    <button
      type="button"
      class={cn(
        'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors',
        boundCount > 0 && 'border-primary/40 bg-primary/5',
      )}
    >
      <Blocks class="size-3.5 shrink-0" />
      <span>
        Skills
        <span class="text-muted-foreground">{boundCount}/{skillStates.length}</span>
      </span>
    </button>
  </Popover.Trigger>

  <Popover.Content class="w-80 p-0" align="start" sideOffset={6}>
    <div class="max-h-80 overflow-y-auto py-1">
      {#each skillStates as skill (skill.path)}
        {@const isExpanded = expandedSkillId === skill.id}
        {@const isLoading = loadingSkillId === skill.id}
        {@const cachedContent = contentCache[skill.id]}
        <div>
          <button
            type="button"
            class={cn(
              'hover:bg-muted flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors',
              isExpanded && 'bg-muted/50',
            )}
            onclick={() => toggleExpand(skill)}
          >
            <span
              class={cn(
                'size-1.5 shrink-0 rounded-full',
                skill.bound ? 'bg-primary' : 'bg-muted-foreground/30',
              )}
            ></span>
            <div class="min-w-0 flex-1">
              <div class="text-foreground truncate text-sm">{skill.name}</div>
              {#if skill.description && !isExpanded}
                <div class="text-muted-foreground truncate text-xs">{skill.description}</div>
              {/if}
            </div>
            <Button
              variant="ghost"
              size="icon-sm"
              class={cn(
                'size-6 shrink-0',
                skill.bound
                  ? 'text-primary hover:text-destructive'
                  : 'text-muted-foreground hover:text-primary',
              )}
              onclick={(e) => {
                e.stopPropagation()
                onToggleSkill?.(skill)
              }}
              title={skill.bound ? 'Unbind skill' : 'Bind skill'}
            >
              {#if skill.bound}
                <Link class="size-3" />
              {:else}
                <Unlink class="size-3" />
              {/if}
            </Button>
          </button>

          {#if isExpanded}
            <div class="border-border bg-muted/30 border-y px-3 py-2">
              {#if skill.description}
                <p class="text-muted-foreground mb-2 text-xs leading-relaxed">
                  {skill.description}
                </p>
              {/if}
              <div
                class="text-muted-foreground mb-1.5 text-[10px] font-medium tracking-wider uppercase"
              >
                SKILL.md
              </div>
              {#if isLoading}
                <div class="space-y-1.5">
                  <Skeleton class="h-3 w-[60%]" />
                  <Skeleton class="h-3 w-[85%]" />
                  <Skeleton class="h-3 w-[45%]" />
                </div>
              {:else if loadError && cachedContent === undefined}
                <p class="text-destructive text-xs">{loadError}</p>
              {:else if cachedContent !== undefined}
                <pre
                  class="text-foreground/80 max-h-48 overflow-y-auto font-mono text-[11px] leading-relaxed break-words whitespace-pre-wrap">{cachedContent ||
                    'No content.'}</pre>
              {/if}
              <div class="text-muted-foreground mt-1.5 truncate font-mono text-[10px]">
                {skill.path}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    {#if skillsSettingsHref}
      <div class="border-border border-t px-3 py-1.5">
        <a
          href={skillsSettingsHref}
          class="text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs transition-colors"
          onclick={() => (open = false)}
        >
          <Settings class="size-3" />
          Manage skills
        </a>
      </div>
    {/if}
  </Popover.Content>
</Popover.Root>
