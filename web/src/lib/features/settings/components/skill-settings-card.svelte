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
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'

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
      const workflowName =
        workflows.find((workflow) => workflow.id === workflowId)?.name ?? 'workflow'
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

  function isBound(workflowId: string) {
    return skill.bound_workflows.some((workflow) => workflow.id === workflowId)
  }
</script>

<article class="space-y-4 rounded-lg border p-4">
  <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
    <div class="space-y-1">
      <div class="flex flex-wrap items-center gap-2">
        <h3 class="font-medium">{skill.name}</h3>
        <span class="rounded-full border px-2 py-0.5 text-xs uppercase">
          {skill.is_builtin ? 'builtin' : 'custom'}
        </span>
        <span class="rounded-full border px-2 py-0.5 text-xs uppercase">
          {skill.is_enabled ? 'enabled' : 'disabled'}
        </span>
      </div>
      <p class="text-muted-foreground text-sm">{skill.description || 'No description.'}</p>
      <p class="text-muted-foreground text-xs">
        Created by {skill.created_by || 'unknown'} at {skill.created_at}
      </p>
      <p class="text-muted-foreground text-xs">
        Bound to:
        {skill.bound_workflows.length > 0
          ? skill.bound_workflows.map((workflow) => workflow.name).join(', ')
          : 'No workflows'}
      </p>
    </div>
    <div class="flex flex-wrap gap-2">
      <Button
        type="button"
        variant="outline"
        onclick={() => (editing ? (editing = false) : void startEditing())}
      >
        {editing ? 'Close Editor' : 'Edit'}
      </Button>
      <Button
        type="button"
        variant="outline"
        onclick={() => void handleToggleEnabled()}
        disabled={busy}
      >
        {skill.is_enabled ? 'Disable' : 'Enable'}
      </Button>
      <Button type="button" variant="outline" onclick={() => void handleDelete()} disabled={busy}>
        Delete
      </Button>
    </div>
  </div>

  <div class="space-y-2">
    <h4 class="text-sm font-medium">Workflow Bindings</h4>
    <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
      {#each workflows as workflow (workflow.id)}
        <label class="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
          <span>{workflow.name}</span>
          <input
            type="checkbox"
            checked={isBound(workflow.id)}
            disabled={busy}
            onchange={(event) =>
              void handleWorkflowBinding(
                workflow.id,
                (event.currentTarget as HTMLInputElement).checked,
              )}
          />
        </label>
      {/each}
    </div>
  </div>

  {#if editing}
    <div class="bg-muted/40 space-y-3 rounded-lg p-3">
      <Input bind:value={editDescription} placeholder="Description override" />
      <Textarea bind:value={editContent} class="min-h-48 font-mono text-sm" />
      <div class="flex gap-2">
        <Button onclick={() => void handleSave()} disabled={busy}>Save Changes</Button>
        <Button type="button" variant="outline" onclick={() => (editing = false)}>Cancel</Button>
      </div>
    </div>
  {/if}
</article>
