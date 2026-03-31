<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { createSkill, listSkills, listWorkflows } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import { ChevronUp, Plus, Search } from '@lucide/svelte'
  import SkillSettingsCard from './skill-settings-card.svelte'

  type SkillFilter = 'all' | 'builtin' | 'custom' | 'disabled'

  let loading = $state(false)
  let workflows = $state<Workflow[]>([])
  let skills = $state<Skill[]>([])
  let query = $state('')
  let filter = $state<SkillFilter>('all')

  let showCreate = $state(false)
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

  const counts = $derived({
    total: skills.length,
    enabled: skills.filter((s) => s.is_enabled).length,
    bound: skills.filter((s) => s.bound_workflows.length > 0).length,
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
      showCreate = false
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

<div class="space-y-5">
  <div class="flex items-start justify-between gap-4">
    <div>
      <h2 class="text-foreground text-base font-semibold">Skills Library</h2>
      <p class="text-muted-foreground mt-1 text-sm">
        Manage reusable control-plane skills, their published versions, rollout state, and workflow
        bindings.
      </p>
    </div>
    {#if !loading}
      <div class="flex shrink-0 items-center gap-3">
        <div class="text-muted-foreground flex items-center gap-2 text-xs">
          <span>{counts.total} skills</span>
          <span class="text-muted-foreground/40">/</span>
          <span>{counts.enabled} enabled</span>
          <span class="text-muted-foreground/40">/</span>
          <span>{counts.bound} bound</span>
        </div>
      </div>
    {/if}
  </div>

  <div class="flex flex-wrap items-center gap-2">
    <div class="relative min-w-48 flex-1">
      <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
      <Input bind:value={query} placeholder="Search skills…" class="pl-9" />
    </div>
    <div class="flex items-center gap-1.5">
      {#each ['all', 'builtin', 'custom', 'disabled'] as item}
        <Button
          type="button"
          size="sm"
          variant={filter === item ? 'secondary' : 'ghost'}
          class="h-8 text-xs capitalize"
          onclick={() => (filter = item as SkillFilter)}
        >
          {item}
        </Button>
      {/each}
    </div>
    <Button size="sm" class="h-8 gap-1.5" onclick={() => (showCreate = !showCreate)}>
      {#if showCreate}
        <ChevronUp class="size-3.5" />
      {:else}
        <Plus class="size-3.5" />
      {/if}
      New Skill
    </Button>
  </div>

  {#if showCreate}
    <section class="bg-muted/40 space-y-3 rounded-lg border p-4">
      <div class="grid gap-3 md:grid-cols-2">
        <Input bind:value={createName} placeholder="Skill name, e.g. deploy-docker" />
        <Input bind:value={createDescription} placeholder="Human-readable description" />
      </div>
      <Textarea bind:value={createContent} class="min-h-32 font-mono text-sm" />
      <div class="flex items-center justify-between">
        <label class="flex items-center gap-2 text-sm">
          <input bind:checked={createEnabled} type="checkbox" />
          Enable immediately
        </label>
        <div class="flex gap-2">
          <Button type="button" variant="ghost" size="sm" onclick={() => (showCreate = false)}>
            Cancel
          </Button>
          <Button size="sm" onclick={() => void handleCreate()} disabled={creating}>
            {creating ? 'Creating…' : 'Create'}
          </Button>
        </div>
      </div>
    </section>
  {/if}

  {#if loading}
    <div class="text-muted-foreground py-8 text-sm">Loading skills…</div>
  {:else if filteredSkills.length === 0}
    <div class="text-muted-foreground rounded-lg border border-dashed py-12 text-center text-sm">
      {skills.length === 0
        ? 'No skills yet. Create one to get started.'
        : 'No skills match your filters.'}
    </div>
  {:else}
    <div class="divide-border divide-y rounded-lg border">
      {#each filteredSkills as skill (skill.id)}
        <SkillSettingsCard {skill} {workflows} onChanged={reloadCurrentProjectSkills} />
      {/each}
    </div>
  {/if}
</div>
