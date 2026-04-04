<script lang="ts">
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
</script>

<div class="border-border bg-card space-y-3 rounded-lg border p-4">
  <div>
    <h4 class="text-sm font-semibold">{title}</h4>
    <p class="text-muted-foreground text-xs">{subtitle}</p>
  </div>
  <div class="space-y-2 text-xs">
    <div>
      <div class="text-muted-foreground">Roles</div>
      <div class="mt-1 flex flex-wrap gap-1">
        {#if roles.length > 0}
          {#each roles as role (role)}
            <code class="bg-muted rounded px-1.5 py-0.5">{role}</code>
          {/each}
        {:else}
          <span class="text-muted-foreground">{emptyRoles}</span>
        {/if}
      </div>
    </div>
    <div>
      <div class="text-muted-foreground">Permissions</div>
      <div class="mt-1 flex flex-wrap gap-1">
        {#if permissions.length > 0}
          {#each permissions as permission (permission)}
            <code class="bg-muted rounded px-1.5 py-0.5">{permission}</code>
          {/each}
        {:else}
          <span class="text-muted-foreground">{emptyPermissions}</span>
        {/if}
      </div>
    </div>
  </div>
</div>
