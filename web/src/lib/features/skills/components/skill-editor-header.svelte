<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Popover from '$ui/popover'
  import { ArrowLeft, Bot, Clock, Power, PowerOff, Save, Trash2 } from '@lucide/svelte'
  import type { Skill } from '$lib/api/contracts'

  type SkillHistoryEntry = {
    id: string
    version: number
    created_by: string
    created_at: string
  }

  let {
    skill,
    busy = false,
    hasDirtyChanges = false,
    assistantOpen = false,
    assistantDisabled = false,
    history = [],
    onNavigateBack,
    onSave,
    onToggleEnabled,
    onDelete,
    onToggleAssistant,
  }: {
    skill: Skill
    busy?: boolean
    hasDirtyChanges?: boolean
    assistantOpen?: boolean
    assistantDisabled?: boolean
    history?: SkillHistoryEntry[]
    onNavigateBack?: () => void
    onSave?: () => void
    onToggleEnabled?: () => void
    onDelete?: () => void
    onToggleAssistant?: () => void
  } = $props()
</script>

<header class="border-border flex shrink-0 items-center justify-between border-b px-4 py-2">
  <div class="flex items-center gap-3">
    <Button variant="ghost" size="sm" class="size-7 p-0" onclick={onNavigateBack}>
      <ArrowLeft class="size-4" />
    </Button>
    <div class="flex items-center gap-2">
      <span
        class="size-2 shrink-0 rounded-full {skill.is_enabled
          ? 'bg-emerald-500'
          : 'bg-muted-foreground/40'}"
      ></span>
      <h1 class="text-foreground text-sm font-semibold">{skill.name}</h1>
      <Badge variant="outline" class="text-[10px] uppercase">
        {skill.is_builtin ? 'builtin' : 'custom'}
      </Badge>
      {#if history.length > 0}
        <Popover.Root>
          <Popover.Trigger>
            <Badge variant="outline" class="hover:bg-muted cursor-pointer text-[10px]">
              v{skill.current_version}
            </Badge>
          </Popover.Trigger>
          <Popover.Content class="w-64 p-0" align="start">
            <div class="border-border border-b px-3 py-1.5">
              <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase"
                >Version History</span
              >
            </div>
            <div class="max-h-48 overflow-y-auto py-1">
              {#each history as item (item.id)}
                <div class="flex items-center gap-2 px-3 py-1 text-xs">
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
          </Popover.Content>
        </Popover.Root>
      {:else}
        <Badge variant="outline" class="text-[10px]">v{skill.current_version}</Badge>
      {/if}
    </div>
  </div>

  <div class="flex items-center gap-1">
    <Button
      size="sm"
      class="h-7 gap-1 px-2 text-xs"
      onclick={onSave}
      disabled={busy || !hasDirtyChanges}
    >
      <Save class="size-3" />
      {busy ? 'Saving\u2026' : 'Save'}
    </Button>
    <Button
      variant="ghost"
      size="sm"
      class="h-7 gap-1 px-2 text-xs"
      title="Toggle fix and verify panel"
      onclick={onToggleAssistant}
      disabled={assistantDisabled}
    >
      <Bot class="size-3.5" />
      <span class={assistantOpen ? 'text-primary' : ''}>Fix &amp; verify</span>
    </Button>
    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
      title={skill.is_enabled ? 'Disable' : 'Enable'}
      onclick={onToggleEnabled}
      disabled={busy}
    >
      {#if skill.is_enabled}
        <PowerOff class="size-3.5" />
      {:else}
        <Power class="size-3.5" />
      {/if}
    </Button>
    <Button
      variant="ghost"
      size="sm"
      class="text-destructive hover:text-destructive size-7 p-0"
      title="Delete skill"
      onclick={onDelete}
      disabled={busy}
    >
      <Trash2 class="size-3.5" />
    </Button>
  </div>
</header>
