<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import {
    bindSkill,
    createSkill,
    deleteSkill,
    disableSkill,
    enableSkill,
    getSkill,
    listSkills,
    listWorkflows,
    unbindSkill,
    updateSkill,
  } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'

  type SkillFilter = 'all' | 'builtin' | 'custom' | 'disabled'

  let loading = $state(false)
  let workflows = $state<Workflow[]>([])
  let skills = $state<Skill[]>([])
  let query = $state('')
  let filter = $state<SkillFilter>('all')

  let createName = $state('')
  let createDescription = $state('')
  let createContent = $state('# New Skill\n\nDescribe the workflow here.\n')
  let createEnabled = $state(true)
  let creating = $state(false)

  let editingSkillId = $state('')
  let editDescription = $state('')
  let editContent = $state('')
  let busySkillId = $state('')

  const filteredSkills = $derived.by(() => {
    const lowered = query.trim().toLowerCase()
    return skills.filter((skill) => {
      if (filter === 'builtin' && !skill.is_builtin) return false
      if (filter === 'custom' && skill.is_builtin) return false
      if (filter === 'disabled' && skill.is_enabled) return false
      if (!lowered) return true
      return [skill.name, skill.description, skill.created_by]
        .join(' ')
        .toLowerCase()
        .includes(lowered)
    })
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      workflows = []
      skills = []
      loading = false
      return
    }

    let cancelled = false
    const load = async () => {
      loading = true
      try {
        const [skillPayload, workflowPayload] = await Promise.all([
          listSkills(projectId),
          listWorkflows(projectId),
        ])
        if (cancelled) return
        skills = skillPayload.skills
        workflows = workflowPayload.workflows
      } catch (caughtError) {
        if (!cancelled) {
          toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to load skills.')
        }
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  async function reloadSkills(projectId: string) {
    skills = (await listSkills(projectId)).skills
  }

  async function handleCreate() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return
    if (!createName.trim()) {
      toastStore.error('Skill name is required.')
      return
    }
    if (!createContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }

    creating = true
    try {
      await createSkill(projectId, {
        name: createName.trim(),
        description: createDescription.trim(),
        content: createContent,
        is_enabled: createEnabled,
      })
      await reloadSkills(projectId)
      createName = ''
      createDescription = ''
      createContent = '# New Skill\n\nDescribe the workflow here.\n'
      createEnabled = true
      toastStore.success('Created skill.')
    } catch (caughtError) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to create skill.')
    } finally {
      creating = false
    }
  }

  async function startEditing(skill: Skill) {
    editingSkillId = skill.id
    editDescription = skill.description
    editContent = ''
    busySkillId = skill.id
    try {
      const payload = await getSkill(skill.id)
      editDescription = payload.skill.description
      editContent = payload.content
    } catch (caughtError) {
      editingSkillId = ''
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to load skill.')
    } finally {
      busySkillId = ''
    }
  }

  async function handleSave(skill: Skill) {
    if (!editContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }
    busySkillId = skill.id
    try {
      await updateSkill(skill.id, {
        description: editDescription.trim(),
        content: editContent,
      })
      const projectId = appStore.currentProject?.id
      if (projectId) {
        await reloadSkills(projectId)
      }
      editingSkillId = ''
      toastStore.success(`Updated ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill.')
    } finally {
      busySkillId = ''
    }
  }

  async function handleToggleEnabled(skill: Skill) {
    busySkillId = skill.id
    try {
      if (skill.is_enabled) {
        await disableSkill(skill.id)
      } else {
        await enableSkill(skill.id)
      }
      const projectId = appStore.currentProject?.id
      if (projectId) {
        await reloadSkills(projectId)
      }
      toastStore.success(`${skill.is_enabled ? 'Disabled' : 'Enabled'} ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill state.',
      )
    } finally {
      busySkillId = ''
    }
  }

  async function handleDelete(skill: Skill) {
    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)
      if (!confirmed) return
    }

    busySkillId = skill.id
    try {
      await deleteSkill(skill.id)
      const projectId = appStore.currentProject?.id
      if (projectId) {
        await reloadSkills(projectId)
      }
      if (editingSkillId === skill.id) {
        editingSkillId = ''
      }
      toastStore.success(`Deleted ${skill.name}.`)
    } catch (caughtError) {
      toastStore.error(caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete skill.')
    } finally {
      busySkillId = ''
    }
  }

  async function handleWorkflowBinding(skill: Skill, workflowId: string, shouldBind: boolean) {
    busySkillId = skill.id
    try {
      if (shouldBind) {
        await bindSkill(skill.id, [workflowId])
      } else {
        await unbindSkill(skill.id, [workflowId])
      }
      const projectId = appStore.currentProject?.id
      if (projectId) {
        await reloadSkills(projectId)
      }
      const workflowName = workflows.find((workflow) => workflow.id === workflowId)?.name ?? 'workflow'
      toastStore.success(`${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update skill binding.',
      )
    } finally {
      busySkillId = ''
    }
  }

  function isBound(skill: Skill, workflowId: string) {
    return skill.bound_workflows.some((workflow) => workflow.id === workflowId)
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Skills Library</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage reusable runtime skills, their rollout state, and workflow bindings.
    </p>
  </div>

  <Separator />

  <section class="grid gap-4 rounded-lg border p-4 lg:grid-cols-[14rem_1fr]">
    <div class="space-y-2">
      <h3 class="text-sm font-medium">Create Skill</h3>
      <p class="text-muted-foreground text-sm">
        New skills are stored in the project skill library and can be bound immediately.
      </p>
    </div>
    <div class="space-y-3">
      <div class="grid gap-3 md:grid-cols-2">
        <Input bind:value={createName} placeholder="deploy-docker" />
        <Input bind:value={createDescription} placeholder="Human-readable description" />
      </div>
      <Textarea bind:value={createContent} class="min-h-40 font-mono text-sm" />
      <label class="flex items-center gap-2 text-sm">
        <input bind:checked={createEnabled} type="checkbox" />
        Enable immediately
      </label>
      <Button onclick={() => void handleCreate()} disabled={creating}>
        {creating ? 'Creating…' : 'Create Skill'}
      </Button>
    </div>
  </section>

  <section class="grid gap-3 md:grid-cols-[1fr_auto]">
    <Input bind:value={query} placeholder="Search skills by name, description, or creator" />
    <div class="flex flex-wrap gap-2">
      {#each ['all', 'builtin', 'custom', 'disabled'] as item}
        <Button
          type="button"
          variant={filter === item ? 'default' : 'outline'}
          onclick={() => (filter = item as SkillFilter)}
        >
          {item}
        </Button>
      {/each}
    </div>
  </section>

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading skills…</div>
  {:else if filteredSkills.length === 0}
    <div class="text-muted-foreground rounded-lg border border-dashed p-6 text-sm">
      No skills matched the current query.
    </div>
  {:else}
    <div class="space-y-4">
      {#each filteredSkills as skill (skill.id)}
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
                onclick={() =>
                  editingSkillId === skill.id ? (editingSkillId = '') : void startEditing(skill)}
              >
                {editingSkillId === skill.id ? 'Close Editor' : 'Edit'}
              </Button>
              <Button
                type="button"
                variant="outline"
                onclick={() => void handleToggleEnabled(skill)}
                disabled={busySkillId === skill.id}
              >
                {skill.is_enabled ? 'Disable' : 'Enable'}
              </Button>
              <Button
                type="button"
                variant="outline"
                onclick={() => void handleDelete(skill)}
                disabled={busySkillId === skill.id}
              >
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
                    checked={isBound(skill, workflow.id)}
                    disabled={busySkillId === skill.id}
                    onchange={(event) =>
                      void handleWorkflowBinding(
                        skill,
                        workflow.id,
                        (event.currentTarget as HTMLInputElement).checked,
                      )}
                  />
                </label>
              {/each}
            </div>
          </div>

          {#if editingSkillId === skill.id}
            <div class="space-y-3 rounded-lg bg-muted/40 p-3">
              <Input bind:value={editDescription} placeholder="Description override" />
              <Textarea bind:value={editContent} class="min-h-48 font-mono text-sm" />
              <div class="flex gap-2">
                <Button onclick={() => void handleSave(skill)} disabled={busySkillId === skill.id}>
                  Save Changes
                </Button>
                <Button type="button" variant="outline" onclick={() => (editingSkillId = '')}>
                  Cancel
                </Button>
              </div>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  {/if}
</div>
