<script lang="ts">
  import type { RoleBinding } from '$lib/api/auth'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import * as Select from '$ui/select'
  import { Plus, Trash2, UserPlus } from '@lucide/svelte'
  import {
    bindingPlaceholder,
    formatTimestamp,
    resolveRoleOption,
    roleOptionsForScope,
    type BindingDraft,
    type RoleOption,
    type ScopeKind,
    type SubjectKind,
  } from './security-settings-human-auth.model'
  import type { TranslationKey } from '$lib/i18n'

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
  const selectedRole = $derived(resolveRoleOption(draft.roleKey))

  const SCOPE_TITLE_KEYS: Record<ScopeKind, TranslationKey> = {
    instance: 'settings.security.humanAuth.scopeTitles.instance',
    organization: 'settings.security.humanAuth.scopeTitles.organization',
    project: 'settings.security.humanAuth.scopeTitles.project',
  }

  const SCOPE_DESCRIPTION_KEYS: Record<ScopeKind, TranslationKey> = {
    instance: 'settings.security.humanAuth.description.instanceScope',
    organization: 'settings.security.humanAuth.description.organizationScope',
    project: 'settings.security.humanAuth.description.projectScope',
  }

  const ROLE_LABEL_KEYS: Record<RoleOption['key'], TranslationKey> = {
    instance_admin: 'settings.security.humanAuth.roles.instanceAdmin.label',
    org_owner: 'settings.security.humanAuth.roles.orgOwner.label',
    org_admin: 'settings.security.humanAuth.roles.orgAdmin.label',
    org_member: 'settings.security.humanAuth.roles.orgMember.label',
    project_admin: 'settings.security.humanAuth.roles.projectAdmin.label',
    project_operator: 'settings.security.humanAuth.roles.projectOperator.label',
    project_reviewer: 'settings.security.humanAuth.roles.projectReviewer.label',
    project_member: 'settings.security.humanAuth.roles.projectMember.label',
    project_viewer: 'settings.security.humanAuth.roles.projectViewer.label',
  }

  const ROLE_SUMMARY_KEYS: Record<RoleOption['key'], TranslationKey> = {
    instance_admin: 'settings.security.humanAuth.roles.instanceAdmin.summary',
    org_owner: 'settings.security.humanAuth.roles.orgOwner.summary',
    org_admin: 'settings.security.humanAuth.roles.orgAdmin.summary',
    org_member: 'settings.security.humanAuth.roles.orgMember.summary',
    project_admin: 'settings.security.humanAuth.roles.projectAdmin.summary',
    project_operator: 'settings.security.humanAuth.roles.projectOperator.summary',
    project_reviewer: 'settings.security.humanAuth.roles.projectReviewer.summary',
    project_member: 'settings.security.humanAuth.roles.projectMember.summary',
    project_viewer: 'settings.security.humanAuth.roles.projectViewer.summary',
  }

  function handleCreate() {
    onCreate?.(scope)
    dialogOpen = false
  }
</script>

<div class="space-y-4">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h4 class="text-sm font-semibold">
            {i18nStore.t(SCOPE_TITLE_KEYS[scope])}
          </h4>
          <p class="text-muted-foreground text-xs">
            {i18nStore.t(SCOPE_DESCRIPTION_KEYS[scope])}
          </p>
        </div>
        {#if canManage}
          <Dialog.Root bind:open={dialogOpen}>
            <Dialog.Trigger>
              {#snippet child({ props })}
                <Button size="sm" {...props}>
                  <Plus class="size-4" />
                  <span class="hidden sm:inline">
                    {i18nStore.t('settings.security.humanAuth.buttons.addBinding')}
                  </span>
                </Button>
              {/snippet}
            </Dialog.Trigger>
        <Dialog.Content class="sm:max-w-md">
          <Dialog.Header>
            <Dialog.Title>
              {i18nStore.t('settings.security.humanAuth.dialogs.addTitle')}
            </Dialog.Title>
            <Dialog.Description>
              {i18nStore.t('settings.security.humanAuth.dialogs.addDescription', {
                scope: i18nStore.t(SCOPE_TITLE_KEYS[scope]),
              })}
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
                <Label>{i18nStore.t('settings.security.humanAuth.labels.subjectType')}</Label>
                <Select.Root
                    type="single"
                    value={draft.subjectKind}
                    onValueChange={(value) =>
                      onSubjectKind?.(scope, (value || 'user') as SubjectKind)}
                  >
                  <Select.Trigger class="w-full">
                    {draft.subjectKind === 'group'
                      ? i18nStore.t('settings.security.humanAuth.options.group')
                      : i18nStore.t('settings.security.humanAuth.options.user')}
                  </Select.Trigger>
                    <Select.Content>
                      <Select.Item value="user">
                        {i18nStore.t('settings.security.humanAuth.options.user')}
                      </Select.Item>
                      <Select.Item value="group">
                        {i18nStore.t('settings.security.humanAuth.options.group')}
                      </Select.Item>
                    </Select.Content>
                  </Select.Root>
                </div>

                <div class="space-y-2">
                  <Label>{i18nStore.t('settings.security.humanAuth.labels.role')}</Label>
                <Select.Root
                  type="single"
                  value={draft.roleKey}
                  onValueChange={(value) =>
                    onRoleKey?.(scope, value || roleOptions[0]?.key || '')}
                >
                  <Select.Trigger class="w-full">
                    {selectedRole
                      ? i18nStore.t(ROLE_LABEL_KEYS[selectedRole.key])
                      : draft.roleKey}
                  </Select.Trigger>
                  <Select.Content>
                    {#each roleOptions as roleOption (roleOption.key)}
                      <Select.Item value={roleOption.key}>
                        <span class="flex flex-col">
                          <span>{i18nStore.t(ROLE_LABEL_KEYS[roleOption.key])}</span>
                        </span>
                      </Select.Item>
                    {/each}
                  </Select.Content>
                </Select.Root>
              </div>
              {#if selectedRole}
                <p class="text-muted-foreground -mt-2 text-xs">
                  {i18nStore.t(ROLE_SUMMARY_KEYS[selectedRole.key])}
                </p>
              {/if}

              <div class="space-y-2">
                <Label for="binding-subject-key">
                  {draft.subjectKind === 'group'
                    ? i18nStore.t('settings.security.humanAuth.labels.groupKey')
                    : i18nStore.t('settings.security.humanAuth.labels.emailOrUserId')}
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
                    ? i18nStore.t('settings.security.humanAuth.hints.groupKey')
                    : i18nStore.t('settings.security.humanAuth.hints.emailOrUserId')}
                </p>
              </div>

              <div class="space-y-2">
                <Label for="binding-expires">
                  {i18nStore.t('settings.security.humanAuth.labels.expiresAt')}
                </Label>
                <Input
                  id="binding-expires"
                  type="datetime-local"
                  value={draft.expiresAtLocal}
                  oninput={(event) =>
                    onExpiresAt?.(scope, (event.currentTarget as HTMLInputElement).value)}
                />
              </div>
              </div>
            </Dialog.Body>

            <Dialog.Footer>
              <Dialog.Close>
                {#snippet child({ props })}
                  <Button variant="outline" {...props}>
                    {i18nStore.t('settings.security.humanAuth.buttons.cancel')}
                  </Button>
                {/snippet}
              </Dialog.Close>
              <Button type="submit" disabled={isCreating}>
                <UserPlus class="size-4" />
                {isCreating
                  ? i18nStore.t('settings.security.humanAuth.buttons.addingBinding')
                  : i18nStore.t('settings.security.humanAuth.buttons.addBinding')}
              </Button>
            </Dialog.Footer>
          </form>
        </Dialog.Content>
      </Dialog.Root>
    {:else}
      <span class="text-muted-foreground text-xs">
        {i18nStore.t('settings.security.humanAuth.messages.readOnly')}
      </span>
    {/if}
  </div>

  {#if bindings.length > 0}
    <div class="divide-border border-border overflow-hidden rounded-lg border">
      {#each bindings as binding, index (binding.id)}
        {@const bindingRole = resolveRoleOption(binding.roleKey)}
        <div
          class="flex flex-col gap-3 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between {index >
          0
            ? 'border-border border-t'
            : ''}"
        >
          <div class="min-w-0 space-y-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-medium">
              {bindingRole
                ? i18nStore.t(ROLE_LABEL_KEYS[bindingRole.key])
                : binding.roleKey}
            </span>
              <code class="bg-muted truncate rounded px-1.5 py-0.5 text-xs">
                {binding.subjectKind}:{binding.subjectKey}
              </code>
            </div>
            <div class="text-muted-foreground text-xs">
              {i18nStore.t('settings.security.humanAuth.messages.grantedBy', {
                grantedBy: binding.grantedBy,
                createdAt: formatTimestamp(binding.createdAt),
              })}
              {#if binding.expiresAt}
                {i18nStore.t('settings.security.humanAuth.messages.expiresAt', {
                  expiresAt: formatTimestamp(binding.expiresAt),
                })}
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
              <span class="sr-only">
                {i18nStore.t('settings.security.humanAuth.buttons.deleteBinding')}
              </span>
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
      <p>{i18nStore.t('settings.security.humanAuth.emptyState.title')}</p>
      {#if canManage}
        <p class="text-xs">
          {i18nStore.t('settings.security.humanAuth.emptyState.hint')}
        </p>
      {/if}
    </div>
  {/if}
</div>
