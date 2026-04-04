<script lang="ts">
  import type { BuiltinRole, BuiltinRoleDetail } from '$lib/api/contracts'
  import { getBuiltinRole, listBuiltinRoles } from '$lib/api/openase'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Sheet from '$ui/sheet'
  import { ArrowLeft, BookTemplate, Loader2, Sparkles } from '@lucide/svelte'
  import type { WorkflowFamily } from '../types'
  import { normalizeWorkflowFamily, workflowFamilyColors, workflowFamilyIcons } from '../model'

  let {
    open = $bindable(false),
    onUseTemplate,
  }: {
    open?: boolean
    onUseTemplate?: (role: BuiltinRole) => void
  } = $props()

  let roles = $state<BuiltinRole[]>([])
  let loading = $state(false)
  let loadingDetail = $state(false)
  let selectedRole = $state<BuiltinRoleDetail | null>(null)
  const roleDetails = new Map<string, BuiltinRoleDetail>()
  let selectRequestId = 0

  function familyOf(role: BuiltinRole | BuiltinRoleDetail): WorkflowFamily {
    return normalizeWorkflowFamily(role.workflow_family ?? '')
  }

  $effect(() => {
    if (!open) {
      selectedRole = null
      return
    }

    if (roles.length > 0) return

    let cancelled = false
    loading = true

    void listBuiltinRoles()
      .then((payload) => {
        if (!cancelled) {
          loading = false
          roles = payload.roles
        }
      })
      .catch(() => {
        if (!cancelled) {
          loading = false
          roles = []
        }
      })

    return () => {
      cancelled = true
    }
  })

  async function handleSelectRole(role: BuiltinRole) {
    const cached = roleDetails.get(role.slug)
    if (cached) {
      selectedRole = cached
      return
    }

    selectedRole = role
    loadingDetail = true
    const requestId = ++selectRequestId

    try {
      const payload = await getBuiltinRole(role.slug)
      if (requestId !== selectRequestId) return

      roleDetails.set(role.slug, payload.role)
      selectedRole = payload.role
    } catch {
      if (requestId === selectRequestId) {
        selectedRole = role
      }
    } finally {
      if (requestId === selectRequestId) {
        loadingDetail = false
      }
    }
  }

  function handleUseTemplate(role: BuiltinRoleDetail) {
    onUseTemplate?.(role)
    open = false
  }
</script>

<Sheet.Root bind:open>
  <Sheet.Content
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-2xl"
    data-testid="workflow-template-gallery"
  >
    <Sheet.Header class="border-border border-b px-6 py-5 text-left">
      {#if selectedRole}
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground -ml-1 flex size-7 items-center justify-center rounded-md transition-colors"
            onclick={() => (selectedRole = null)}
          >
            <ArrowLeft class="size-4" />
          </button>
          <div class="flex-1">
            <Sheet.Title>{selectedRole.name}</Sheet.Title>
            <Sheet.Description class="mt-0.5">{selectedRole.summary}</Sheet.Description>
          </div>
        </div>
      {:else}
        <div class="flex items-center gap-2">
          <BookTemplate class="text-muted-foreground size-5" />
          <Sheet.Title>Workflow Templates</Sheet.Title>
        </div>
        <Sheet.Description>
          Browse preset workflow roles and use them as a starting point.
        </Sheet.Description>
      {/if}
    </Sheet.Header>

    <div class="flex-1 overflow-y-auto">
      {#if loading}
        <div class="text-muted-foreground flex items-center justify-center py-20 text-sm">
          Loading templates…
        </div>
      {:else if selectedRole}
        <!-- Detail view -->
        <div class="space-y-4 p-6">
          {#if loadingDetail}
            <div class="text-muted-foreground flex items-center gap-2 py-8 text-sm">
              <Loader2 class="size-4 animate-spin" />
              Loading template details…
            </div>
          {/if}

          <div class="flex items-center gap-2">
            <Badge
              variant="outline"
              class={cn('text-[10px]', workflowFamilyColors[familyOf(selectedRole)])}
            >
              {selectedRole.workflow_type}
            </Badge>
            <span class="text-muted-foreground text-xs">{selectedRole.harness_path}</span>
          </div>

          <div
            class="border-border bg-muted/30 overflow-x-auto rounded-lg border p-4 font-mono text-sm leading-relaxed whitespace-pre-wrap"
          >
            {selectedRole.workflow_content || selectedRole.content}
          </div>
        </div>
      {:else}
        <!-- Grid view -->
        <div class="grid gap-3 p-6 sm:grid-cols-2">
          {#each roles as role (role.slug)}
            {@const workflowFamily = familyOf(role)}
            <button
              type="button"
              class="border-border hover:border-foreground/20 hover:bg-muted/50 group flex flex-col gap-2 rounded-xl border p-4 text-left transition-all"
              onclick={() => void handleSelectRole(role)}
            >
              <div class="flex items-center gap-2">
                <span class="text-base">{workflowFamilyIcons[workflowFamily]}</span>
                <span class="text-foreground flex-1 text-sm font-medium">{role.name}</span>
                <Badge
                  variant="outline"
                  class={cn('text-[10px]', workflowFamilyColors[workflowFamily])}
                >
                  {role.workflow_type}
                </Badge>
              </div>
              <p class="text-muted-foreground line-clamp-2 text-xs leading-relaxed">
                {role.summary}
              </p>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    {#if selectedRole}
      <div class="border-border flex items-center justify-end gap-2 border-t px-6 py-4">
        <Button variant="outline" size="sm" onclick={() => (selectedRole = null)}>Back</Button>
        <Button size="sm" onclick={() => handleUseTemplate(selectedRole!)} disabled={loadingDetail}>
          <Sparkles class="size-3.5" />
          Use this template
        </Button>
      </div>
    {/if}
  </Sheet.Content>
</Sheet.Root>
