<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { SecuritySettingsResponse } from '$lib/api/contracts'
  import { updateProject } from '$lib/api/openase'
  import { ScopeGroupPicker, type ScopeGroup } from '$lib/features/workflows'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'

  type Security = SecuritySettingsResponse['security']

  let { security }: { security: Security } = $props()

  let saving = $state(false)
  let draftText = $state('')
  let selectedScopes = $state<string[]>([])

  const projectAIScopeGroups = $derived(
    filterProjectAIScopeGroups(security.agent_tokens.supported_scope_groups),
  )
  const availableScopes = $derived(projectAIScopeGroups.flatMap((group) => group.scopes))
  const baselineScopes = $derived(
    normalizeProjectAIScopes(
      appStore.currentProject?.project_ai_platform_access_allowed,
      availableScopes,
    ),
  )
  const dirty = $derived(arrayKey(selectedScopes) !== arrayKey(baselineScopes))

  $effect(() => {
    selectedScopes = [...baselineScopes]
    draftText = baselineScopes.join('\n')
  })

  function filterProjectAIScopeGroups(
    groups: Array<{ category: string; scopes: string[] }> | undefined,
  ): ScopeGroup[] {
    return (groups ?? [])
      .map((group) => ({
        category: group.category,
        scopes: group.scopes.filter((scope) => scope !== 'tickets.update.self'),
      }))
      .filter((group) => group.scopes.length > 0)
  }

  function normalizeProjectAIScopes(
    scopes: string[] | null | undefined,
    fallback: string[],
  ): string[] {
    const source = Array.isArray(scopes) && scopes.length > 0 ? scopes : fallback
    const allowed = new Set(fallback)
    return Array.from(
      new Set(
        source
          .map((scope) => scope.trim())
          .filter((scope) => scope !== '' && (allowed.size === 0 || allowed.has(scope))),
      ),
    ).sort()
  }

  function arrayKey(scopes: string[]): string {
    return [...scopes].sort().join(',')
  }

  function handleScopeChange(scopes: string[]) {
    selectedScopes = normalizeProjectAIScopes(scopes, availableScopes)
    draftText = selectedScopes.join('\n')
  }

  function handleTextChange(value: string) {
    draftText = value
    selectedScopes = normalizeProjectAIScopes(
      value
        .split(/[\n,]/)
        .map((scope) => scope.trim())
        .filter(Boolean),
      availableScopes,
    )
  }

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    saving = true
    try {
      const payload = await updateProject(projectId, {
        project_ai_platform_access_allowed: [...selectedScopes],
      })
      appStore.currentProject = payload.project
      selectedScopes = normalizeProjectAIScopes(
        payload.project.project_ai_platform_access_allowed,
        availableScopes,
      )
      draftText = selectedScopes.join('\n')
      toastStore.success('Project AI access saved.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save Project AI access.',
      )
    } finally {
      saving = false
    }
  }
</script>

<div class="space-y-4">
  <div>
    <h3 class="text-sm font-semibold">Project AI platform access</h3>
    <p class="text-muted-foreground mt-1 text-xs">
      Reuses the workflow scope picker, but applies to Project AI conversations at the project
      level. New and existing projects default to the full Project AI scope set.
    </p>
  </div>

  <div class="space-y-1.5">
    <Label class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
      >Platform Access Allowed</Label
    >
    {#if projectAIScopeGroups.length > 0}
      <ScopeGroupPicker
        groups={projectAIScopeGroups}
        selected={selectedScopes}
        disabled={saving}
        onchange={handleScopeChange}
      />
    {:else}
      <Textarea
        value={draftText}
        rows={10}
        class="min-h-36 font-mono text-xs"
        placeholder="One scope per line, e.g. tickets.list"
        disabled={saving}
        oninput={(event) => handleTextChange((event.currentTarget as HTMLTextAreaElement).value)}
      />
    {/if}
  </div>

  <div class="flex justify-start">
    <Button onclick={handleSave} disabled={saving || !dirty}>
      {saving ? 'Saving…' : 'Save Project AI access'}
    </Button>
  </div>
</div>
