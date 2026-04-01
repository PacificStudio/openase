<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    bindSkill,
    deleteSkill,
    disableSkill,
    enableSkill,
    getSkill,
    listSkillHistory,
    unbindSkill,
    updateSkill,
  } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Pencil, Power, PowerOff, Trash2 } from '@lucide/svelte'
  import SkillDetailBody from './skill-detail-body.svelte'

  let {
    open = $bindable(false),
    skill,
    workflows = [],
    onChanged,
    onDeleted,
  }: {
    open?: boolean
    skill: Skill | null
    workflows?: Workflow[]
    onChanged?: () => Promise<void> | void
    onDeleted?: (skillId: string) => void
  } = $props()

  let editing = $state(false)
  let editDescription = $state('')
  let editContent = $state('')
  let content = $state('')
  let history = $state<
    Array<{ id: string; version: number; created_by: string; created_at: string }>
  >([])
  let busy = $state(false)
  let loadingDetail = $state(false)
  let loaded = $state(false)

  $effect(() => {
    if (open && skill && !loaded) {
      void loadDetail(skill.id)
    }
    if (!open) {
      editing = false
      loaded = false
      content = ''
      history = []
    }
  })

  async function loadDetail(skillId: string) {
    loadingDetail = true
    try {
      const [detailPayload, historyPayload] = await Promise.all([
        getSkill(skillId),
        listSkillHistory(skillId),
      ])
      content = detailPayload.content
      history = historyPayload.history
      loaded = true
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to load skill detail.')
    } finally {
      loadingDetail = false
    }
  }

  function startEditing() {
    if (!skill) return
    editDescription = skill.description
    editContent = content
    editing = true
  }

  function cancelEditing() {
    editing = false
  }

  async function handleSave() {
    if (!skill) return
    if (!editContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }
    busy = true
    try {
      await updateSkill(skill.id, {
        description: editDescription.trim(),
        content: editContent,
      })
      content = editContent
      editing = false
      toastStore.success(`Updated ${skill.name}.`)
      await onChanged?.()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill.')
    } finally {
      busy = false
    }
  }

  async function handleToggleEnabled() {
    if (!skill) return
    busy = true
    try {
      if (skill.is_enabled) {
        await disableSkill(skill.id)
      } else {
        await enableSkill(skill.id)
      }
      toastStore.success(`${skill.is_enabled ? 'Disabled' : 'Enabled'} ${skill.name}.`)
      await onChanged?.()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill state.')
    } finally {
      busy = false
    }
  }

  async function handleDelete() {
    if (!skill) return
    const confirmed = window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)
    if (!confirmed) return

    busy = true
    try {
      await deleteSkill(skill.id)
      toastStore.success(`Deleted ${skill.name}.`)
      onDeleted?.(skill.id)
      open = false
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete skill.')
    } finally {
      busy = false
    }
  }

  async function handleWorkflowBinding(workflowId: string, shouldBind: boolean) {
    if (!skill) return
    busy = true
    try {
      if (shouldBind) {
        await bindSkill(skill.id, [workflowId])
      } else {
        await unbindSkill(skill.id, [workflowId])
      }
      const workflowName = workflows.find((w) => w.id === workflowId)?.name ?? 'workflow'
      toastStore.success(
        `${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`,
      )
      await onChanged?.()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill binding.')
    } finally {
      busy = false
    }
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl">
    {#if !skill}
      <SheetHeader class="p-6">
        <SheetTitle>Skill</SheetTitle>
      </SheetHeader>
    {:else}
      <!-- Header -->
      <SheetHeader class="border-border shrink-0 border-b px-6 py-4 text-left">
        <div class="flex items-center justify-between gap-3 pr-10">
          <div class="flex min-w-0 items-center gap-2.5">
            <span
              class="mt-0.5 size-2 shrink-0 rounded-full {skill.is_enabled
                ? 'bg-emerald-500'
                : 'bg-muted-foreground/40'}"
            ></span>
            <SheetTitle class="truncate text-base">{skill.name}</SheetTitle>
            <Badge variant="outline" class="shrink-0 text-[10px] uppercase">
              {skill.is_builtin ? 'builtin' : 'custom'}
            </Badge>
            <Badge variant="outline" class="shrink-0 text-[10px]">
              v{skill.current_version}
            </Badge>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            {#if !editing}
              <Button
                variant="ghost"
                size="sm"
                class="size-7 p-0"
                title={editing ? 'Cancel edit' : 'Edit skill'}
                onclick={startEditing}
                disabled={busy || loadingDetail}
              >
                <Pencil class="size-3.5" />
              </Button>
            {/if}
            <Button
              variant="ghost"
              size="sm"
              class="size-7 p-0"
              title={skill.is_enabled ? 'Disable' : 'Enable'}
              onclick={() => void handleToggleEnabled()}
              disabled={busy}
            >
              {#if skill.is_enabled}
                <PowerOff class="size-3.5" />
              {:else}
                <Power class="size-3.5" />
              {/if}
            </Button>
          </div>
        </div>
      </SheetHeader>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto">
        <SkillDetailBody
          {skill}
          {workflows}
          {loadingDetail}
          {editing}
          {busy}
          {content}
          bind:editDescription
          bind:editContent
          {history}
          onCancelEdit={cancelEditing}
          onSave={() => void handleSave()}
          onToggleBinding={handleWorkflowBinding}
        />
      </div>

      <!-- Footer -->
      <div class="border-border shrink-0 border-t px-6 py-3">
        <Button
          variant="ghost"
          size="sm"
          class="text-destructive hover:text-destructive gap-1.5"
          disabled={busy}
          onclick={() => void handleDelete()}
        >
          <Trash2 class="size-3.5" />
          Delete skill
        </Button>
      </div>
    {/if}
  </SheetContent>
</Sheet>
