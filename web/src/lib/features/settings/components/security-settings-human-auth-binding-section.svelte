<script lang="ts">
  import type { RoleBinding } from '$lib/api/auth'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import {
    bindingPlaceholder,
    formatTimestamp,
    resolveRoleOption,
    roleOptions,
    scopeTitle,
    type BindingDraft,
    type ScopeKind,
    type SubjectKind,
  } from './security-settings-human-auth.model'

  let {
    scope,
    bindings = [],
    canManage = false,
    draft,
    mutationKey = '',
    onSubjectKind,
    onSubjectKey,
    onRoleKey,
    onExpiresAt,
    onCreate,
    onDelete,
  }: {
    scope: ScopeKind
    bindings?: RoleBinding[]
    canManage?: boolean
    draft: BindingDraft
    mutationKey?: string
    onSubjectKind?: (scope: ScopeKind, value: SubjectKind) => void
    onSubjectKey?: (scope: ScopeKind, value: string) => void
    onRoleKey?: (scope: ScopeKind, value: string) => void
    onExpiresAt?: (scope: ScopeKind, value: string) => void
    onCreate?: (scope: ScopeKind) => void
    onDelete?: (scope: ScopeKind, bindingId: string) => void
  } = $props()
</script>

<div class="border-border bg-card space-y-4 rounded-lg border p-4">
  <div class="flex items-start justify-between gap-3">
    <div>
      <h4 class="text-sm font-semibold">{scopeTitle(scope)}</h4>
      <p class="text-muted-foreground text-xs">
        {scope === 'organization'
          ? 'Bindings here inherit into descendant projects.'
          : 'Project-scoped roles stack with direct and group bindings.'}
      </p>
    </div>
    <div class="text-muted-foreground text-xs">{canManage ? 'Editable' : 'Read only'}</div>
  </div>

  {#if canManage}
    <div class="grid gap-2 lg:grid-cols-[9rem_minmax(0,1fr)_13rem_13rem_auto]">
      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Subject kind</span>
        <select
          class="border-input bg-background h-9 rounded-md border px-3 text-sm"
          value={draft.subjectKind}
          onchange={(event) =>
            onSubjectKind?.(scope, (event.currentTarget as HTMLSelectElement).value as SubjectKind)}
        >
          <option value="user">User</option>
          <option value="group">Group</option>
        </select>
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Subject key</span>
        <Input
          value={draft.subjectKey}
          oninput={(event) =>
            onSubjectKey?.(scope, (event.currentTarget as HTMLInputElement).value)}
          placeholder={bindingPlaceholder(draft.subjectKind)}
        />
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Role</span>
        <select
          class="border-input bg-background h-9 rounded-md border px-3 text-sm"
          value={draft.roleKey}
          onchange={(event) => onRoleKey?.(scope, (event.currentTarget as HTMLSelectElement).value)}
        >
          {#each roleOptions as roleOption (roleOption.key)}
            <option value={roleOption.key}>{roleOption.label}</option>
          {/each}
        </select>
      </label>

      <label class="space-y-1 text-xs">
        <span class="text-muted-foreground">Expires at</span>
        <Input
          type="datetime-local"
          value={draft.expiresAtLocal}
          oninput={(event) => onExpiresAt?.(scope, (event.currentTarget as HTMLInputElement).value)}
        />
      </label>

      <div class="flex items-end">
        <Button disabled={mutationKey === `${scope}:create`} onclick={() => onCreate?.(scope)}>
          {mutationKey === `${scope}:create` ? 'Adding…' : 'Add binding'}
        </Button>
      </div>
    </div>

    <p class="text-muted-foreground text-xs">
      Use a user email or stable user id for direct grants. Group subject keys should match the
      synchronized OIDC group key.
    </p>
  {:else}
    <p class="text-muted-foreground text-xs">
      <code>rbac.manage</code> is required on this scope before bindings can be edited.
    </p>
  {/if}

  <div class="space-y-2">
    {#if bindings.length > 0}
      {#each bindings as binding (binding.id)}
        <div
          class="border-border bg-muted/20 flex flex-col gap-3 rounded-lg border px-4 py-3 text-xs lg:flex-row lg:items-start lg:justify-between"
        >
          <div class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-medium"
                >{resolveRoleOption(binding.roleKey)?.label ?? binding.roleKey}</span
              >
              <code class="bg-background rounded px-1.5 py-0.5">
                {binding.subjectKind}:{binding.subjectKey}
              </code>
            </div>
            <div class="text-muted-foreground">
              {resolveRoleOption(binding.roleKey)?.summary ?? 'Static builtin role'}
            </div>
            <div class="text-muted-foreground">
              Granted by {binding.grantedBy} · created {formatTimestamp(binding.createdAt)}
              {#if binding.expiresAt}
                · expires {formatTimestamp(binding.expiresAt)}
              {/if}
            </div>
          </div>

          {#if canManage}
            <Button
              variant="outline"
              size="sm"
              disabled={mutationKey === `${scope}:delete:${binding.id}`}
              onclick={() => onDelete?.(scope, binding.id)}
            >
              {mutationKey === `${scope}:delete:${binding.id}` ? 'Deleting…' : 'Delete'}
            </Button>
          {/if}
        </div>
      {/each}
    {:else}
      <div class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-sm">
        No bindings yet on this scope.
      </div>
    {/if}
  </div>
</div>
