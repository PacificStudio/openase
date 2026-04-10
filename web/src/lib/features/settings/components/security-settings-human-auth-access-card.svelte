<script lang="ts">
  import { Badge } from '$ui/badge'
  import * as Collapsible from '$ui/collapsible'
  import { ChevronRight, Shield } from '@lucide/svelte'

  let {
    title,
    subtitle = '',
    roles = [],
    permissions = [],
    emptyRoles = 'No roles',
    emptyPermissions = 'No permissions',
  }: {
    title: string
    subtitle?: string
    roles?: string[]
    permissions?: string[]
    emptyRoles?: string
    emptyPermissions?: string
  } = $props()

  let permissionsOpen = $state(false)

  const groupedPermissions = $derived.by(() => {
    const groups: Record<string, string[]> = {}
    for (const perm of permissions) {
      const dotIndex = perm.indexOf('.')
      const domain = dotIndex > 0 ? perm.slice(0, dotIndex) : 'other'
      ;(groups[domain] ??= []).push(perm)
    }
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b))
  })
</script>

<div class="border-border bg-card space-y-3 rounded-lg border p-4">
  <div class="flex items-start gap-2">
    <Shield class="text-muted-foreground mt-0.5 size-4 shrink-0" />
    <div class="min-w-0">
      <h4 class="text-sm font-semibold">{title}</h4>
      {#if subtitle}
        <p class="text-muted-foreground text-xs">{subtitle}</p>
      {/if}
    </div>
  </div>

  <div class="space-y-2 text-xs">
    <div>
      <div class="text-muted-foreground mb-1.5 text-xs font-medium">Roles</div>
      <div class="flex flex-wrap gap-1.5">
        {#if roles.length > 0}
          {#each roles as role (role)}
            <Badge variant="outline" class="font-mono text-xs">{role}</Badge>
          {/each}
        {:else}
          <span class="text-muted-foreground">{emptyRoles}</span>
        {/if}
      </div>
    </div>

    <Collapsible.Root bind:open={permissionsOpen}>
      <Collapsible.Trigger>
        {#snippet child({ props })}
          <button
            {...props}
            type="button"
            class="text-muted-foreground hover:text-foreground mt-1 flex items-center gap-1 text-xs transition-colors"
          >
            <ChevronRight
              class="size-3.5 transition-transform {permissionsOpen ? 'rotate-90' : ''}"
            />
            {permissions.length} permissions
          </button>
        {/snippet}
      </Collapsible.Trigger>
      <Collapsible.Content>
        <div class="mt-2 space-y-2">
          {#if permissions.length > 0}
            {#each groupedPermissions as [domain, perms] (domain)}
              <div>
                <div class="text-muted-foreground mb-1 text-xs font-medium capitalize">
                  {domain}
                </div>
                <div class="flex flex-wrap gap-1">
                  {#each perms as permission (permission)}
                    <code class="bg-muted rounded px-1.5 py-0.5 text-xs">{permission}</code>
                  {/each}
                </div>
              </div>
            {/each}
          {:else}
            <span class="text-muted-foreground">{emptyPermissions}</span>
          {/if}
        </div>
      </Collapsible.Content>
    </Collapsible.Root>
  </div>
</div>
