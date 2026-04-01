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
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Textarea } from '$ui/textarea'
  import {
    Clock,
    Link2,
    Link2Off,
    Pencil,
    Power,
    PowerOff,
    Save,
    Trash2,
    X,
  } from '@lucide/svelte'

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
    const confirmed = window.confirm(
      `Delete "${skill.name}" and remove it from all workflows?`,
    )
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

  function isBound(workflowId: string) {
    return skill?.bound_workflows.some((w) => w.id === workflowId) ?? false
  }
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl"
  >
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
              class="mt-0.5 size-2 shrink-0 rounded-full {skill.is_enabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'}"
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
        {#if loadingDetail}
          <div class="text-muted-foreground px-6 py-10 text-center text-sm">Loading…</div>
        {:else}
          <!-- Meta section -->
          <section class="border-border space-y-3 border-b px-6 py-4">
            {#if editing}
              <div class="space-y-1.5">
                <span class="text-muted-foreground text-[11px] font-medium uppercase tracking-wider">
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
                <Badge
                  variant="secondary"
                  class="text-muted-foreground px-1.5 py-0.5 text-[10px]"
                >
                  disabled
                </Badge>
              {/if}
            </div>
          </section>

          <!-- Workflow bindings -->
          <section class="border-border space-y-2 border-b px-6 py-4">
            <h3 class="text-muted-foreground text-[11px] font-medium uppercase tracking-wider">
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
                    onclick={() => void handleWorkflowBinding(workflow.id, !bound)}
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

          <!-- Content -->
          <section class="border-border space-y-2 border-b px-6 py-4">
            <div class="flex items-center justify-between">
              <h3 class="text-muted-foreground text-[11px] font-medium uppercase tracking-wider">
                SKILL.md
              </h3>
              {#if editing}
                <div class="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="h-7 gap-1 px-2 text-xs"
                    onclick={cancelEditing}
                    disabled={busy}
                  >
                    <X class="size-3" />
                    Cancel
                  </Button>
                  <Button
                    size="sm"
                    class="h-7 gap-1 px-2 text-xs"
                    onclick={() => void handleSave()}
                    disabled={busy}
                  >
                    <Save class="size-3" />
                    {busy ? 'Saving…' : 'Publish'}
                  </Button>
                </div>
              {/if}
            </div>
            {#if editing}
              <Textarea
                bind:value={editContent}
                class="min-h-64 font-mono text-sm"
                disabled={busy}
              />
            {:else}
              <pre
                class="bg-muted/40 max-h-96 overflow-auto rounded-lg border p-4 text-sm leading-relaxed whitespace-pre-wrap"
              >{content || '(empty)'}</pre>
            {/if}
          </section>

          <!-- Version history -->
          {#if history.length > 0}
            <section class="space-y-2 px-6 py-4">
              <h3 class="text-muted-foreground text-[11px] font-medium uppercase tracking-wider">
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
