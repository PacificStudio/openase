<script lang="ts">
  import type { RoleBinding } from '$lib/api/auth'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Plus, Trash2, UserPlus } from '@lucide/svelte'
  import {
    bindingPlaceholder,
    formatTimestamp,
    resolveRoleOption,
    roleOptionsForScope,
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

  let dialogOpen = $state(false)

  const roleOptions = $derived(roleOptionsForScope(scope))
  const isCreating = $derived(mutationKey === `${scope}:create`)

  function handleCreate() {
    onCreate?.(scope)
    dialogOpen = false
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h4 class="text-sm font-semibold">{scopeTitle(scope)}</h4>
      <p class="text-muted-foreground text-xs">
        {scope === 'instance'
          ? 'Instance-scoped grants apply everywhere and should stay tightly controlled.'
          : scope === 'organization'
            ? 'Bindings here inherit into descendant projects.'
            : 'Project-scoped roles stack with direct and group bindings.'}
      </p>
    </div>
    {#if canManage}
      <Dialog.Root bind:open={dialogOpen}>
        <Dialog.Trigger>
          {#snippet child({ props })}
            <Button size="sm" {...props}>
              <Plus class="size-4" />
              <span class="hidden sm:inline">Add binding</span>
            </Button>
          {/snippet}
        </Dialog.Trigger>
        <Dialog.Content class="sm:max-w-md">
          <Dialog.Header>
            <Dialog.Title>Add role binding</Dialog.Title>
            <Dialog.Description>
              Grant a user or group access to this {scope}.
            </Dialog.Description>
          </Dialog.Header>

          <form
            class="flex min-h-0 flex-1 flex-col gap-6"
            onsubmit={(event) => {
              event.preventDefault()
              handleCreate()
            }}
          >
            <Dialog.Body class="space-y-4">
              <div class="grid gap-4 sm:grid-cols-2">
                <div class="space-y-2">
                  <Label>Subject type</Label>
                  <Select.Root
                    type="single"
                    value={draft.subjectKind}
                    onValueChange={(value) =>
                      onSubjectKind?.(scope, (value || 'user') as SubjectKind)}
                  >
                    <Select.Trigger class="w-full">
                      {draft.subjectKind === 'group' ? 'Group' : 'User'}
                    </Select.Trigger>
                    <Select.Content>
                      <Select.Item value="user">User</Select.Item>
                      <Select.Item value="group">Group</Select.Item>
                    </Select.Content>
                  </Select.Root>
                </div>

                <div class="space-y-2">
                  <Label>Role</Label>
                  <Select.Root
                    type="single"
                    value={draft.roleKey}
                    onValueChange={(value) =>
                      onRoleKey?.(scope, value || roleOptions[0]?.key || '')}
                  >
                    <Select.Trigger class="w-full">
                      {resolveRoleOption(draft.roleKey)?.label ?? draft.roleKey}
                    </Select.Trigger>
                    <Select.Content>
                      {#each roleOptions as roleOption (roleOption.key)}
                        <Select.Item value={roleOption.key}>
                          <span class="flex flex-col">
                            <span>{roleOption.label}</span>
                          </span>
                        </Select.Item>
                      {/each}
                    </Select.Content>
                  </Select.Root>
                </div>
              </div>

              {#if resolveRoleOption(draft.roleKey)?.summary}
                <p class="text-muted-foreground -mt-2 text-xs">
                  {resolveRoleOption(draft.roleKey)?.summary}
                </p>
              {/if}

              <div class="space-y-2">
                <Label for="binding-subject-key">
                  {draft.subjectKind === 'group' ? 'Group key' : 'Email or user ID'}
                </Label>
                <Input
                  id="binding-subject-key"
                  value={draft.subjectKey}
                  oninput={(event) =>
                    onSubjectKey?.(scope, (event.currentTarget as HTMLInputElement).value)}
                  placeholder={bindingPlaceholder(draft.subjectKind)}
                />
                <p class="text-muted-foreground text-xs">
                  {draft.subjectKind === 'group'
                    ? 'Must match the synchronized OIDC group key.'
                    : 'Use a stable email or user ID for direct grants.'}
                </p>
              </div>

              <div class="space-y-2">
                <Label for="binding-expires">Expires at (optional)</Label>
                <Input
                  id="binding-expires"
                  type="datetime-local"
                  value={draft.expiresAtLocal}
                  oninput={(event) =>
                    onExpiresAt?.(scope, (event.currentTarget as HTMLInputElement).value)}
                />
              </div>
            </Dialog.Body>

            <Dialog.Footer>
              <Dialog.Close>
                {#snippet child({ props })}
                  <Button variant="outline" {...props}>Cancel</Button>
                {/snippet}
              </Dialog.Close>
              <Button type="submit" disabled={isCreating}>
                <UserPlus class="size-4" />
                {isCreating ? 'Adding...' : 'Add binding'}
              </Button>
            </Dialog.Footer>
          </form>
        </Dialog.Content>
      </Dialog.Root>
    {:else}
      <span class="text-muted-foreground text-xs">Read only</span>
    {/if}
  </div>

  {#if bindings.length > 0}
    <div class="divide-border border-border overflow-hidden rounded-lg border">
      {#each bindings as binding, index (binding.id)}
        <div
          class="flex flex-col gap-3 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between {index >
          0
            ? 'border-border border-t'
            : ''}"
        >
          <div class="min-w-0 space-y-0.5">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-medium">
                {resolveRoleOption(binding.roleKey)?.label ?? binding.roleKey}
              </span>
              <code class="bg-muted truncate rounded px-1.5 py-0.5 text-xs">
                {binding.subjectKind}:{binding.subjectKey}
              </code>
            </div>
            <div class="text-muted-foreground text-xs">
              Granted by {binding.grantedBy} · {formatTimestamp(binding.createdAt)}
              {#if binding.expiresAt}
                · expires {formatTimestamp(binding.expiresAt)}
              {/if}
            </div>
          </div>

          {#if canManage}
            <Button
              variant="ghost"
              size="icon-sm"
              class="text-muted-foreground hover:text-destructive shrink-0 self-end sm:self-auto"
              disabled={mutationKey === `${scope}:delete:${binding.id}`}
              onclick={() => onDelete?.(scope, binding.id)}
            >
              <Trash2 class="size-4" />
              <span class="sr-only">Delete binding</span>
            </Button>
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <div
      class="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm"
    >
      <UserPlus class="text-muted-foreground/50 size-8" />
      <p>No bindings on this scope yet.</p>
      {#if canManage}
        <p class="text-xs">Click "Add binding" to grant access.</p>
      {/if}
    </div>
  {/if}
</div>
