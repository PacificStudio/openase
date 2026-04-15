<script lang="ts">
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { createSkill, listSkills } from '$lib/api/openase'
  import type { Skill } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Skeleton } from '$ui/skeleton'
  import { ChevronRight, Link2, Plus, Search, Wrench } from '@lucide/svelte'
  import PageHelpButton from '$lib/components/layout/page-help-button.svelte'
  import SkillsPageCreateSheet from './skills-page-create-sheet.svelte'
  import { skillsT } from './i18n'
  import type { TranslationKey } from '$lib/i18n'

  type SkillFilter = 'all' | 'builtin' | 'custom' | 'disabled'

  const filterValues: SkillFilter[] = ['all', 'builtin', 'custom', 'disabled']
  const filterLabelKeys: Record<SkillFilter, TranslationKey> = {
    all: 'skills.filter.all',
    builtin: 'skills.filter.builtin',
    custom: 'skills.filter.custom',
    disabled: 'skills.filter.disabled',
  }
  const statsLabelKeys: Record<'total' | 'enabled' | 'bound', TranslationKey> = {
    total: 'skills.stats.total',
    enabled: 'skills.stats.enabled',
    bound: 'skills.stats.bound',
  }
  const statusTitleKeys: Record<'enabled' | 'disabled', TranslationKey> = {
    enabled: 'skills.status.enabled',
    disabled: 'skills.status.disabled',
  }
  const badgeLabelKeys: Record<'builtin' | 'custom' | 'disabled', TranslationKey> = {
    builtin: 'skills.badge.builtin',
    custom: 'skills.badge.custom',
    disabled: 'skills.badge.disabled',
  }

  let loading = $state(false)
  let skills = $state<Skill[]>([])
  let query = $state('')
  let filter = $state<SkillFilter>('all')
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
      skills = []
      loading = false
      return
    }

    let cancelled = false
    const load = async () => {
      loading = true
      try {
        const skillPayload = await listSkills(projectId)
        if (cancelled) return
        skills = skillPayload.skills
      } catch (caughtError) {
        if (!cancelled) {
          toastStore.error(
            caughtError instanceof ApiError ? caughtError.detail : skillsT('skills.loadFailed'),
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

  function openSkill(skill: Skill) {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (!orgId || !projectId) return
    void goto(`${projectPath(orgId, projectId, 'skills')}/${skill.id}`)
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
      toastStore.error(skillsT('skills.validation.nameRequired'))
      return
    }
    if (!createContent.trim()) {
      toastStore.error(skillsT('skills.validation.contentRequired'))
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
      const payload = await listSkills(projectId)
      skills = payload.skills
      createOpen = false
      toastStore.success(skillsT('skills.createSuccess'))
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : skillsT('skills.createFailed'),
      )
    } finally {
      creating = false
    }
  }
</script>

<div data-testid="route-scroll-container" class="min-h-0 flex-1 overflow-y-auto">
  <div class="mx-auto max-w-5xl space-y-5 p-6">
    <div class="flex items-start justify-between gap-4">
      <div>
        <div class="flex items-center gap-1.5">
          <h1 class="text-foreground text-xl font-semibold">
            {skillsT('skills.page.title')}
          </h1>
          <PageHelpButton section="skills" />
        </div>
        <p class="text-muted-foreground mt-1 text-sm">
          {skillsT('skills.page.description')}
        </p>
      </div>
      {#if !loading}
        <div
          class="text-muted-foreground flex items-center gap-2 text-xs"
          data-tour="skills-stats-region"
        >
          <span>{skillsT(statsLabelKeys.total, { count: counts.total })}</span>
          <span class="text-muted-foreground/40">/</span>
          <span>{skillsT(statsLabelKeys.enabled, { count: counts.enabled })}</span>
          <span class="text-muted-foreground/40">/</span>
          <span>{skillsT(statsLabelKeys.bound, { count: counts.bound })}</span>
        </div>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-2" data-tour="skills-toolbar">
      <div class="relative min-w-48 flex-1">
        <Search class="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input bind:value={query} placeholder={skillsT('skills.searchPlaceholder')} class="pl-9" />
      </div>
      <div class="flex items-center gap-1.5">
        {#each filterValues as item}
          <Button
            type="button"
            size="sm"
            variant={filter === item ? 'secondary' : 'ghost'}
            class="h-8 text-xs capitalize"
            onclick={() => (filter = item as SkillFilter)}
          >
            {skillsT(filterLabelKeys[item])}
          </Button>
        {/each}
      </div>
      <Button size="sm" class="h-8 gap-1.5" data-tour="skills-create" onclick={openCreate}>
        <Plus class="size-3.5" />
        {skillsT('skills.actions.newSkill')}
      </Button>
    </div>

    {#if loading}
      <div class="divide-border divide-y rounded-lg border">
        {#each { length: 5 } as _}
          <div class="flex items-start gap-3 px-4 py-3">
            <Skeleton class="mt-1.5 size-2 shrink-0 rounded-full" />
            <div class="min-w-0 flex-1 space-y-2">
              <div class="flex items-center gap-2">
                <Skeleton class="h-4 w-32" />
                <Skeleton class="h-4 w-12 rounded-full" />
                <Skeleton class="h-4 w-8 rounded-full" />
              </div>
              <Skeleton class="h-3 w-48" />
            </div>
            <Skeleton class="mt-0.5 h-3.5 w-16" />
          </div>
        {/each}
      </div>
    {:else if filteredSkills.length === 0}
      <div class="animate-fade-in-up rounded-lg border border-dashed py-14 text-center">
        {#if skills.length === 0}
          <div
            class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full"
          >
            <Wrench class="text-muted-foreground size-5" />
          </div>
          <p class="text-foreground text-sm font-medium">
            {skillsT('skills.empty.title')}
          </p>
          <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-sm">
            {skillsT('skills.empty.description')}
          </p>
        {:else}
          <p class="text-muted-foreground text-sm">
            {skillsT('skills.empty.filteredTitle')}
          </p>
        {/if}
      </div>
    {:else}
      <div class="divide-border divide-y rounded-lg border" data-tour="skills-list-grid">
        {#each filteredSkills as skill, idx (skill.id)}
          {@const boundCount = skill.bound_workflows.length}
          {@const boundNames = skill.bound_workflows.map((w) => w.name).join(', ')}
          <button
            type="button"
            class="group hover:bg-muted/40 animate-stagger flex w-full items-start gap-3 px-4 py-3 text-left transition-colors"
            style="--stagger-index: {idx}"
            onclick={() => openSkill(skill)}
          >
            <span
              class="mt-1.5 size-2 shrink-0 rounded-full {skill.is_enabled
                ? 'animate-pulse-dot bg-emerald-500'
                : 'bg-muted-foreground/40'}"
              title={skillsT(skill.is_enabled ? statusTitleKeys.enabled : statusTitleKeys.disabled)}
            ></span>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-foreground text-sm font-medium">{skill.name}</span>
                <Badge variant="outline" class="px-1.5 py-0 text-[10px] leading-relaxed uppercase">
                  {skillsT(skill.is_builtin ? badgeLabelKeys.builtin : badgeLabelKeys.custom)}
                </Badge>
                <Badge variant="outline" class="px-1.5 py-0 text-[10px] leading-relaxed">
                  v{skill.current_version}
                </Badge>
                {#if !skill.is_enabled}
                  <Badge
                    variant="secondary"
                    class="text-muted-foreground px-1.5 py-0 text-[10px] leading-relaxed"
                  >
                    {skillsT(badgeLabelKeys.disabled)}
                  </Badge>
                {/if}
              </div>

              {#if skill.description}
                <p class="text-muted-foreground mt-0.5 truncate text-xs">{skill.description}</p>
              {/if}

              {#if boundCount > 0}
                <div class="text-muted-foreground mt-1 flex items-center gap-1 text-[11px]">
                  <Link2 class="size-3 shrink-0" />
                  <span class="truncate">{boundNames}</span>
                </div>
              {/if}
            </div>

            <div class="flex shrink-0 items-center gap-2">
              <span class="text-muted-foreground text-xs">
                {skill.is_builtin ? '' : skill.created_by}
              </span>
              <ChevronRight
                class="text-muted-foreground/40 group-hover:text-foreground/60 size-4 shrink-0 transition-colors"
              />
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>

<SkillsPageCreateSheet
  bind:open={createOpen}
  bind:name={createName}
  bind:description={createDescription}
  bind:content={createContent}
  bind:enabled={createEnabled}
  {creating}
  onCreate={() => void handleCreate()}
/>
