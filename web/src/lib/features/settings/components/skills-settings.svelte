<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { createSkill, listSkills, listWorkflows } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'
  import SkillSettingsCard from './skill-settings-card.svelte'

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
  const currentProjectId = $derived(appStore.currentProject?.id)

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
          toastStore.error(
            caughtError instanceof ApiError ? caughtError.detail : 'Failed to load skills.',
          )
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

  async function reloadCurrentProjectSkills() {
    if (!currentProjectId) return
    await reloadSkills(currentProjectId)
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
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create skill.',
      )
    } finally {
      creating = false
    }
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
        <SkillSettingsCard {skill} {workflows} onChanged={reloadCurrentProjectSkills} />
      {/each}
    </div>
  {/if}
</div>
