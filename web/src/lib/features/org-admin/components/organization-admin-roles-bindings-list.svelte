<script lang="ts">
  import type { RoleBinding } from '$lib/api/auth'
  import { formatTimestamp, resolveRoleOption } from '$lib/features/org-admin/role-bindings'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    loading,
    error,
    bindings,
    canManageBindings,
    canDeleteBinding,
    mutationKey,
    onDeleteBinding,
  }: {
    loading: boolean
    error: string
    bindings: RoleBinding[]
    canManageBindings: boolean
    canDeleteBinding: (binding: RoleBinding) => boolean
    mutationKey: string
    onDeleteBinding: (binding: RoleBinding) => void
  } = $props()
  const t = i18nStore.t
</script>

{#if error}
  <div
    class="border-destructive/30 bg-destructive/5 text-destructive rounded-md border px-4 py-3 text-sm"
  >
    {error}
  </div>
{/if}

{#if loading}
  <div class="text-muted-foreground rounded-md border border-dashed px-4 py-8 text-center text-sm">
    {t('orgAdmin.roles.bindings.loading')}
  </div>
{:else if bindings.length === 0}
  <div class="text-muted-foreground rounded-md border border-dashed px-4 py-8 text-center text-sm">
    {t('orgAdmin.roles.bindings.empty')}
  </div>
{:else}
  <div class="border-border bg-card rounded-md border">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b">
          <th class="text-muted-foreground px-4 py-2.5 text-left text-xs font-medium">
            {t('orgAdmin.roles.bindings.columns.subject')}
          </th>
          <th class="text-muted-foreground px-4 py-2.5 text-left text-xs font-medium">
            {t('orgAdmin.roles.bindings.columns.role')}
          </th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium md:table-cell"
          >
            {t('orgAdmin.roles.bindings.columns.grantedBy')}
          </th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium lg:table-cell"
          >
            {t('orgAdmin.roles.bindings.columns.created')}
          </th>
          <th
            class="text-muted-foreground hidden px-4 py-2.5 text-left text-xs font-medium lg:table-cell"
          >
            {t('orgAdmin.roles.bindings.columns.expires')}
          </th>
          <th class="px-4 py-2.5"></th>
        </tr>
      </thead>
      <tbody>
        {#each bindings as binding (binding.id)}
          <tr class="border-b last:border-0">
            <td class="px-4 py-3">
              <div class="flex flex-wrap items-center gap-2">
                <Badge variant={binding.subjectKind === 'group' ? 'secondary' : 'outline'}>
                  {binding.subjectKind}
                </Badge>
                <code class="bg-muted rounded px-1.5 py-0.5 text-xs">{binding.subjectKey}</code>
              </div>
            </td>
            <td class="px-4 py-3 font-medium">
              {resolveRoleOption(binding.roleKey)?.label ?? binding.roleKey}
            </td>
            <td class="text-muted-foreground hidden px-4 py-3 md:table-cell">
              {binding.grantedBy}
            </td>
            <td class="text-muted-foreground hidden px-4 py-3 lg:table-cell">
              {formatTimestamp(binding.createdAt)}
            </td>
            <td class="text-muted-foreground hidden px-4 py-3 lg:table-cell">
              {binding.expiresAt ? formatTimestamp(binding.expiresAt) : '—'}
            </td>
            <td class="px-4 py-3 text-right">
              {#if canDeleteBinding(binding)}
                <Button
                  variant="ghost"
                  size="sm"
                  onclick={() => onDeleteBinding(binding)}
                  disabled={mutationKey === `delete:${binding.id}`}
                >
                  {mutationKey === `delete:${binding.id}`
                    ? t('orgAdmin.roles.bindings.actions.deleting')
                    : t('orgAdmin.roles.bindings.actions.delete')}
                </Button>
              {:else if canManageBindings}
                <span class="text-muted-foreground text-xs">
                  {t('orgAdmin.roles.bindings.ownerRequired')}
                </span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
