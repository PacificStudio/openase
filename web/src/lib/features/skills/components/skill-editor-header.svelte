<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import {
    ArrowLeft,
    Power,
    PowerOff,
    Save,
    Trash2,
    PanelRightOpen,
    PanelRightClose,
  } from '@lucide/svelte'
  import type { Skill } from '$lib/api/contracts'

  let {
    skill,
    busy = false,
    hasDirtyChanges = false,
    metadataOpen = true,
    onNavigateBack,
    onSave,
    onToggleEnabled,
    onDelete,
    onToggleMetadata,
  }: {
    skill: Skill
    busy?: boolean
    hasDirtyChanges?: boolean
    metadataOpen?: boolean
    onNavigateBack?: () => void
    onSave?: () => void
    onToggleEnabled?: () => void
    onDelete?: () => void
    onToggleMetadata?: () => void
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
      <Badge variant="outline" class="text-[10px]">v{skill.current_version}</Badge>
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
    <div class="bg-border mx-1 h-4 w-px"></div>
    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
      title={metadataOpen ? 'Hide metadata' : 'Show metadata'}
      onclick={onToggleMetadata}
    >
      {#if metadataOpen}
        <PanelRightClose class="size-3.5" />
      {:else}
        <PanelRightOpen class="size-3.5" />
      {/if}
    </Button>
  </div>
</header>
