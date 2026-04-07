<script lang="ts">
  import type { RoleBinding } from '$lib/api/auth'
  import { formatTimestamp, resolveRoleOption } from '$lib/features/org-admin/role-bindings'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'

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
</script>

{#if error}
  <div
    class="border-destructive/30 bg-destructive/5 text-destructive rounded-2xl border px-4 py-3 text-sm"
  >
    {error}
  </div>
{/if}

{#if loading}
  <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
    Loading org role bindings…
  </div>
{:else if bindings.length === 0}
  <div class="text-muted-foreground rounded-2xl border border-dashed px-4 py-8 text-sm">
    No org role bindings yet.
  </div>
{:else}
  <div class="space-y-3">
    {#each bindings as binding (binding.id)}
      <div class="rounded-3xl border bg-white p-4 shadow-sm">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div class="space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant={binding.subjectKind === 'group' ? 'secondary' : 'outline'}>
                {binding.subjectKind}
              </Badge>
              <span class="font-medium"
                >{resolveRoleOption(binding.roleKey)?.label ?? binding.roleKey}</span
              >
              <code class="rounded-full bg-slate-100 px-3 py-1 text-xs">
                {binding.subjectKey}
              </code>
            </div>
            <div class="text-muted-foreground text-sm leading-6">
              {resolveRoleOption(binding.roleKey)?.summary ?? 'Builtin org role'}
            </div>
            <div class="text-muted-foreground text-xs">
              Granted by {binding.grantedBy} · created {formatTimestamp(binding.createdAt)}
              {#if binding.expiresAt}
                · expires {formatTimestamp(binding.expiresAt)}
              {/if}
            </div>
          </div>

          {#if canDeleteBinding(binding)}
            <Button
              variant="outline"
              size="sm"
              onclick={() => onDeleteBinding(binding)}
              disabled={mutationKey === `delete:${binding.id}`}
            >
              {mutationKey === `delete:${binding.id}` ? 'Deleting…' : 'Delete'}
            </Button>
          {:else if canManageBindings}
            <div class="text-muted-foreground text-xs">
              Owner approval required to change privileged bindings.
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
{/if}
