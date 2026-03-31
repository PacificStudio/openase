<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    bindSkill,
    deleteSkill,
    disableSkill,
    enableSkill,
    getSkill,
    unbindSkill,
    updateSkill,
  } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { Ellipsis, Link2, Link2Off, Pencil, Power, PowerOff, Trash2 } from '@lucide/svelte'

  let {
    skill,
    workflows,
    onChanged,
  }: {
    skill: Skill
    workflows: Workflow[]
    onChanged: () => Promise<void> | void
  } = $props()

  let editing = $state(false)
  let editDescription = $state('')
  let editContent = $state('')
  let busy = $state(false)

  function isBound(workflowId: string) {
    return skill.bound_workflows.some((w) => w.id === workflowId)
  }

  async function startEditing() {
    editing = true
    editDescription = skill.description
    editContent = ''
    busy = true
    try {
      const payload = await getSkill(skill.id)
      editDescription = payload.skill.description
      editContent = payload.content
    } catch (caughtError) {
      editing = false
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load skill.',
      )
    } finally {
      busy = false
    }
  }

  async function handleSave() {
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
      await onChanged()
      editing = false
      toastStore.success(`Updated ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill.',
      )
    } finally {
      busy = false
    }
  }

  async function handleToggleEnabled() {
    busy = true
    try {
      if (skill.is_enabled) {
        await disableSkill(skill.id)
      } else {
        await enableSkill(skill.id)
      }
      await onChanged()
      toastStore.success(`${skill.is_enabled ? 'Disabled' : 'Enabled'} ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill state.',
      )
    } finally {
      busy = false
    }
  }

  async function handleDelete() {
    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)
      if (!confirmed) return
    }

    busy = true
    try {
      await deleteSkill(skill.id)
      await onChanged()
      editing = false
      toastStore.success(`Deleted ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete skill.',
      )
    } finally {
      busy = false
    }
  }

  async function handleWorkflowBinding(workflowId: string, shouldBind: boolean) {
    busy = true
    try {
      if (shouldBind) {
        await bindSkill(skill.id, [workflowId])
      } else {
        await unbindSkill(skill.id, [workflowId])
      }
      await onChanged()
      const workflowName = workflows.find((w) => w.id === workflowId)?.name ?? 'workflow'
      toastStore.success(
        `${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`,
      )
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill binding.',
      )
    } finally {
      busy = false
    }
  }
</script>

<article class="group hover:bg-muted/30 px-4 py-3 transition-colors">
  <div class="flex items-start gap-3">
    <!-- Status indicator -->
    <div class="mt-1.5 flex shrink-0 items-center">
      <span
        class="size-2 rounded-full {skill.is_enabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'}"
        title={skill.is_enabled ? 'Enabled' : 'Disabled'}
      ></span>
    </div>

    <!-- Main content -->
    <div class="min-w-0 flex-1 space-y-1.5">
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium">{skill.name}</span>
        <Badge variant="outline" class="px-1.5 py-0.5 text-[10px] leading-none uppercase">
          {skill.is_builtin ? 'builtin' : 'custom'}
        </Badge>
        {#if !skill.is_enabled}
          <Badge
            variant="secondary"
            class="text-muted-foreground px-1.5 py-0.5 text-[10px] leading-none"
          >
            disabled
          </Badge>
        {/if}
      </div>

      {#if skill.description}
        <p class="text-muted-foreground text-xs leading-relaxed">{skill.description}</p>
      {/if}

      <!-- Workflow bindings -->
      <div class="flex flex-wrap items-center gap-1.5">
        {#if workflows.length > 0}
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
        {:else}
          <span class="text-muted-foreground text-xs">No workflows available</span>
        {/if}
      </div>
    </div>

    <!-- Actions -->
    <div class="flex shrink-0 items-center gap-1">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger>
          {#snippet child({ props })}
            <Button {...props} variant="ghost" size="sm" class="size-7 p-0">
              <Ellipsis class="size-4" />
            </Button>
          {/snippet}
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="end" class="w-40">
          <DropdownMenu.Item onclick={() => void startEditing()} disabled={busy}>
            <Pencil class="mr-2 size-3.5" />
            Edit
          </DropdownMenu.Item>
          <DropdownMenu.Item onclick={() => void handleToggleEnabled()} disabled={busy}>
            {#if skill.is_enabled}
              <PowerOff class="mr-2 size-3.5" />
              Disable
            {:else}
              <Power class="mr-2 size-3.5" />
              Enable
            {/if}
          </DropdownMenu.Item>
          <DropdownMenu.Separator />
          <DropdownMenu.Item
            class="text-destructive focus:text-destructive"
            onclick={() => void handleDelete()}
            disabled={busy}
          >
            <Trash2 class="mr-2 size-3.5" />
            Delete
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Root>
    </div>
  </div>

  {#if editing}
    <div class="bg-muted/40 mt-3 ml-5 space-y-3 rounded-lg border p-3">
      <Input bind:value={editDescription} placeholder="Description" class="text-sm" />
      <Textarea bind:value={editContent} class="min-h-40 font-mono text-sm" />
      <div class="flex justify-end gap-2">
        <Button type="button" variant="ghost" size="sm" onclick={() => (editing = false)}>
          Cancel
        </Button>
        <Button size="sm" onclick={() => void handleSave()} disabled={busy}>Save</Button>
      </div>
    </div>
  {/if}
</article>
