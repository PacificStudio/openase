<script lang="ts">
  import type { Skill } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Link2 } from '@lucide/svelte'

  let {
    skill,
    onSelect,
  }: {
    skill: Skill
    onSelect?: (skill: Skill) => void
  } = $props()

  const boundCount = $derived(skill.bound_workflows.length)
  const boundNames = $derived(skill.bound_workflows.map((w) => w.name).join(', '))
</script>

<button
  type="button"
  class="hover:bg-muted/40 flex w-full items-start gap-3 px-4 py-3 text-left transition-colors"
  onclick={() => onSelect?.(skill)}
>
  <span
    class="mt-1.5 size-2 shrink-0 rounded-full {skill.is_enabled
      ? 'bg-emerald-500'
      : 'bg-muted-foreground/40'}"
    title={skill.is_enabled ? 'Enabled' : 'Disabled'}
  ></span>

  <div class="min-w-0 flex-1">
    <div class="flex items-center gap-2">
      <span class="text-foreground text-sm font-medium">{skill.name}</span>
      <Badge variant="outline" class="px-1.5 py-0 text-[10px] leading-relaxed uppercase">
        {skill.is_builtin ? 'builtin' : 'custom'}
      </Badge>
      <Badge variant="outline" class="px-1.5 py-0 text-[10px] leading-relaxed">
        v{skill.current_version}
      </Badge>
      {#if !skill.is_enabled}
        <Badge
          variant="secondary"
          class="text-muted-foreground px-1.5 py-0 text-[10px] leading-relaxed"
        >
          disabled
        </Badge>
      {/if}
    </div>

    {#if skill.description}
      <p class="text-muted-foreground mt-0.5 truncate text-xs">{skill.description}</p>
    {/if}

    {#if boundCount > 0}
      <div class="text-muted-foreground mt-1 flex items-center gap-1 text-[11px]">
        <Link2 class="size-3 shrink-0" />
        <span class="truncate">{boundNames}</span>
      </div>
    {/if}
  </div>
</button>
