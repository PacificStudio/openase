<script lang="ts">
  import type { EffectivePermissionsResponse, RoleBinding } from '$lib/api/auth'
  import { OrganizationMembersSection } from '$lib/features/organizations'
  import type { HumanAuthUser } from '$lib/stores/auth.svelte'
  import { LockKeyhole, Users } from '@lucide/svelte'
  import SecuritySettingsHumanAuthAccessCard from './security-settings-human-auth-access-card.svelte'
  import SecuritySettingsHumanAuthBindingSection from './security-settings-human-auth-binding-section.svelte'
  import SecuritySettingsHumanAuthSessions from './security-settings-human-auth-sessions.svelte'
  import SecuritySettingsUserDirectory from './security-settings-user-directory.svelte'
  import type { BindingDraft, ScopeKind, SubjectKind } from './security-settings-human-auth.model'
  import { i18nStore } from '$lib/i18n/store.svelte'

  type GroupSummary = {
    group_key: string
    group_name: string
    issuer: string
  }

  type ApprovalPoliciesSummary = {
    status: string
    rules_count: number
    summary: string
  }

  let {
    user = null,
    currentOrgId = '',
    currentOrgName = '',
    currentProjectName = '',
    currentGroups = [],
    approvalPolicies = null,
    error = '',
    instancePermissions = null,
    orgPermissions = null,
    projectPermissions = null,
    instanceBindings = [],
    orgBindings = [],
    projectBindings = [],
    canManageInstanceBindings = false,
    canManageOrgBindings = false,
    canManageProjectBindings = false,
    instanceDraft,
    orgDraft,
    projectDraft,
    mutationKey = '',
    onDraftSubjectKind,
    onDraftSubjectKey,
    onDraftRoleKey,
    onDraftExpiresAt,
    onCreateBinding,
    onDeleteBinding,
  }: {
    user?: HumanAuthUser | null
    currentOrgId?: string
    currentOrgName?: string
    currentProjectName?: string
    currentGroups?: GroupSummary[]
    approvalPolicies?: ApprovalPoliciesSummary | null
    error?: string
    instancePermissions?: EffectivePermissionsResponse | null
    orgPermissions?: EffectivePermissionsResponse | null
    projectPermissions?: EffectivePermissionsResponse | null
    instanceBindings?: RoleBinding[]
    orgBindings?: RoleBinding[]
    projectBindings?: RoleBinding[]
    canManageInstanceBindings?: boolean
    canManageOrgBindings?: boolean
    canManageProjectBindings?: boolean
    instanceDraft: BindingDraft
    orgDraft: BindingDraft
    projectDraft: BindingDraft
    mutationKey?: string
    onDraftSubjectKind?: (scope: ScopeKind, value: SubjectKind) => void
    onDraftSubjectKey?: (scope: ScopeKind, value: string) => void
    onDraftRoleKey?: (scope: ScopeKind, value: string) => void
    onDraftExpiresAt?: (scope: ScopeKind, value: string) => void
    onCreateBinding?: (scope: ScopeKind) => void
    onDeleteBinding?: (scope: ScopeKind, bindingId: string) => void
  } = $props()

  const canReadUserDirectory = $derived(
    instancePermissions?.permissions.includes('security_setting.read') ?? false,
  )
  const canManageUserDirectory = $derived(
    instancePermissions?.permissions.includes('security_setting.update') ?? false,
  )
</script>

<div class="grid gap-4 xl:grid-cols-2">
  <div class="border-border bg-card space-y-3 rounded-lg border p-4">
    <div class="flex items-center gap-2">
      <Users class="text-muted-foreground size-4" />
      <h4 class="text-sm font-semibold">
        {i18nStore.t('settings.security.humanAuth.headings.identity')}
      </h4>
    </div>
    <div class="grid gap-3 text-xs sm:grid-cols-2">
      <div>
        <div class="text-muted-foreground">
          {i18nStore.t('settings.security.humanAuth.labels.userId')}
        </div>
        <div class="mt-1 font-mono break-all">{user?.id ?? ''}</div>
      </div>
      <div>
        <div class="text-muted-foreground">
          {i18nStore.t('settings.security.humanAuth.labels.groups')}
        </div>
        <div class="mt-1 flex flex-wrap gap-1">
          {#if currentGroups.length > 0}
            {#each currentGroups as group (group.issuer + ':' + group.group_key)}
              <code class="bg-muted rounded px-1.5 py-0.5">
                {group.group_name || group.group_key}
              </code>
            {/each}
          {:else}
            <span class="text-muted-foreground">
              {i18nStore.t('settings.security.humanAuth.messages.noSynchronizedGroups')}
            </span>
          {/if}
        </div>
      </div>
    </div>
  </div>

  <div class="border-border bg-card space-y-3 rounded-lg border p-4">
    <div class="flex items-center gap-2">
      <LockKeyhole class="text-muted-foreground size-4" />
      <h4 class="text-sm font-semibold">
        {i18nStore.t('settings.security.humanAuth.headings.approvalBoundary')}
      </h4>
    </div>
    <p class="text-muted-foreground text-xs">
      {approvalPolicies?.summary ??
        i18nStore.t('settings.security.humanAuth.description.approvalSummary')}
    </p>
    <div class="grid gap-3 text-xs sm:grid-cols-2">
      <div>
        <div class="text-muted-foreground">
          {i18nStore.t('settings.security.humanAuth.labels.policyStatus')}
        </div>
        <div class="mt-1 font-medium uppercase">
          {approvalPolicies?.status ??
            i18nStore.t('settings.security.humanAuth.fallbacks.policyReserved')}
        </div>
      </div>
      <div>
        <div class="text-muted-foreground">
          {i18nStore.t('settings.security.humanAuth.labels.storedRules')}
        </div>
        <div class="mt-1 font-medium">{approvalPolicies?.rules_count ?? 0}</div>
      </div>
    </div>
    <div class="text-muted-foreground text-xs">
      {i18nStore.t('settings.security.humanAuth.description.auditAttribution')}
    </div>
  </div>
</div>

<div class="grid gap-4 xl:grid-cols-3">
  <SecuritySettingsHumanAuthAccessCard
    title={i18nStore.t('settings.security.humanAuth.accessCardTitles.instance')}
    subtitle={i18nStore.t('settings.security.humanAuth.accessCardSubtitles.controlPlane')}
    roles={instancePermissions?.roles ?? []}
    permissions={instancePermissions?.permissions ?? []}
    emptyRoles={i18nStore.t('settings.security.humanAuth.messages.noInstanceRoles')}
    emptyPermissions={i18nStore.t('settings.security.humanAuth.messages.noInstancePermissions')}
  />

  <SecuritySettingsHumanAuthAccessCard
    title={i18nStore.t('settings.security.humanAuth.accessCardTitles.organization')}
    subtitle={currentOrgName}
    roles={orgPermissions?.roles ?? []}
    permissions={orgPermissions?.permissions ?? []}
    emptyRoles={i18nStore.t('settings.security.humanAuth.messages.noOrganizationRoles')}
    emptyPermissions={i18nStore.t('settings.security.humanAuth.messages.noOrganizationPermissions')}
  />

  <SecuritySettingsHumanAuthAccessCard
    title={i18nStore.t('settings.security.humanAuth.accessCardTitles.project')}
    subtitle={currentProjectName}
    roles={projectPermissions?.roles ?? []}
    permissions={projectPermissions?.permissions ?? []}
    emptyRoles={i18nStore.t('settings.security.humanAuth.messages.noProjectRoles')}
    emptyPermissions={i18nStore.t('settings.security.humanAuth.messages.noProjectPermissions')}
  />
</div>

<div class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-xs">
  {i18nStore.t('settings.security.humanAuth.description.humanPermissions')}
</div>

{#if error}
  <div class="text-destructive text-sm">{error}</div>
{/if}

<SecuritySettingsHumanAuthSessions />

<SecuritySettingsUserDirectory canRead={canReadUserDirectory} canManage={canManageUserDirectory} />

<SecuritySettingsHumanAuthBindingSection
  scope="instance"
  bindings={instanceBindings}
  canManage={canManageInstanceBindings}
  draft={instanceDraft}
  {mutationKey}
  onSubjectKind={onDraftSubjectKind}
  onSubjectKey={onDraftSubjectKey}
  onRoleKey={onDraftRoleKey}
  onExpiresAt={onDraftExpiresAt}
  onCreate={onCreateBinding}
  onDelete={onDeleteBinding}
/>

<SecuritySettingsHumanAuthBindingSection
  scope="organization"
  bindings={orgBindings}
  canManage={canManageOrgBindings}
  draft={orgDraft}
  {mutationKey}
  onSubjectKind={onDraftSubjectKind}
  onSubjectKey={onDraftSubjectKey}
  onRoleKey={onDraftRoleKey}
  onExpiresAt={onDraftExpiresAt}
  onCreate={onCreateBinding}
  onDelete={onDeleteBinding}
/>

<SecuritySettingsHumanAuthBindingSection
  scope="project"
  bindings={projectBindings}
  canManage={canManageProjectBindings}
  draft={projectDraft}
  {mutationKey}
  onSubjectKind={onDraftSubjectKind}
  onSubjectKey={onDraftSubjectKey}
  onRoleKey={onDraftRoleKey}
  onExpiresAt={onDraftExpiresAt}
  onCreate={onCreateBinding}
  onDelete={onDeleteBinding}
/>

{#if currentOrgId}
  <OrganizationMembersSection organizationId={currentOrgId} />
{/if}
