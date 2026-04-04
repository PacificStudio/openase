<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { createSkill, listSkills, listWorkflows } from '$lib/api/openase'
  import type { Skill, Workflow } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Textarea } from '$ui/textarea'
  import { Plus, Search } from '@lucide/svelte'
  import SkillDetailSheet from './skill-detail-sheet.svelte'
  import SkillSettingsCard from './skill-settings-card.svelte'

  type SkillFilter = 'all' | 'builtin' | 'custom' | 'disabled'

  let loading = $state(false)
  let workflows = $state<Workflow[]>([])
  let skills = $state<Skill[]>([])
  let query = $state('')
  let filter = $state<SkillFilter>('all')

  // Detail sheet
  let detailOpen = $state(false)
  let selectedSkill = $state<Skill | null>(null)

  // Create sheet
  let createOpen = $state(false)
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

  async function reloadSkills() {
    if (!currentProjectId) return
    const payload = await listSkills(currentProjectId)
    skills = payload.skills
    // Refresh selectedSkill if it's still open
    if (selectedSkill) {
      const updated = skills.find((s) => s.id === selectedSkill!.id)
      selectedSkill = updated ?? null
    }
  }

  function openDetail(skill: Skill) {
    selectedSkill = skill
    detailOpen = true
  }

  function openCreate() {
    createName = ''
    createDescription = ''
    createContent = '# New Skill\n\nDescribe the workflow here.\n'
    createEnabled = true
    createOpen = true
  }

  async function handleCreate() {
    const projectId = currentProjectId
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
      await reloadSkills()
      createOpen = false
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
        Manage reusable skills, versions, and workflow bindings.
      </p>
    </div>
    {#if !loading}
      <div class="text-muted-foreground flex items-center gap-2 text-xs">
        <span>{counts.total} skills</span>
        <span class="text-muted-foreground/40">/</span>
        <span>{counts.enabled} enabled</span>
        <span class="text-muted-foreground/40">/</span>
        <span>{counts.bound} bound</span>
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
    <Button size="sm" class="h-8 gap-1.5" onclick={openCreate}>
      <Plus class="size-3.5" />
      New Skill
    </Button>
  </div>

  {#if loading}
    <div class="divide-border divide-y rounded-lg border">
      {#each { length: 4 } as _}
        <div class="flex items-center gap-3 px-4 py-3">
          <div class="min-w-0 flex-1 space-y-1.5">
            <div class="flex items-center gap-2">
              <div class="bg-muted h-4 w-28 animate-pulse rounded"></div>
              <div class="bg-muted h-4 w-14 animate-pulse rounded-full"></div>
            </div>
            <div class="bg-muted h-3 w-48 animate-pulse rounded"></div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
            <div class="bg-muted size-4 animate-pulse rounded"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if filteredSkills.length === 0}
    <div class="text-muted-foreground rounded-lg border border-dashed py-12 text-center text-sm">
      {skills.length === 0
        ? 'No skills yet. Create one to get started.'
        : 'No skills match your filters.'}
    </div>
  {:else}
    <div class="divide-border divide-y rounded-lg border">
      {#each filteredSkills as skill (skill.id)}
        <SkillSettingsCard {skill} onSelect={openDetail} />
      {/each}
    </div>
  {/if}
</div>

<!-- Detail / Edit sheet -->
<SkillDetailSheet
  bind:open={detailOpen}
  skill={selectedSkill}
  {workflows}
  onChanged={reloadSkills}
  onDeleted={() => void reloadSkills()}
/>

<!-- Create sheet -->
<Sheet bind:open={createOpen}>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="border-border shrink-0 border-b px-6 py-4 text-left">
      <div class="flex items-center justify-between gap-4 pr-10">
        <SheetTitle class="text-base">New Skill</SheetTitle>
        <Button size="sm" onclick={() => void handleCreate()} disabled={creating}>
          {creating ? 'Creating…' : 'Create'}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 space-y-4 overflow-y-auto px-6 py-5">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            Name
          </span>
          <Input
            bind:value={createName}
            placeholder="deploy-docker"
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            Description
          </span>
          <Input
            bind:value={createDescription}
            placeholder="Human-readable description"
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
      </div>

      <div class="space-y-1.5">
        <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
          SKILL.md
        </span>
        <Textarea
          bind:value={createContent}
          class="min-h-64 font-mono text-sm"
          disabled={creating}
        />
      </div>

      <label class="flex items-center gap-2 text-sm">
        <input bind:checked={createEnabled} type="checkbox" disabled={creating} />
        Enable immediately
      </label>
    </div>
  </SheetContent>
</Sheet>
