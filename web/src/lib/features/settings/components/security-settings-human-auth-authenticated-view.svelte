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
      <h4 class="text-sm font-semibold">Identity</h4>
    </div>
    <div class="grid gap-3 text-xs sm:grid-cols-2">
      <div>
        <div class="text-muted-foreground">User ID</div>
        <div class="mt-1 font-mono break-all">{user?.id ?? ''}</div>
      </div>
      <div>
        <div class="text-muted-foreground">Groups</div>
        <div class="mt-1 flex flex-wrap gap-1">
          {#if currentGroups.length > 0}
            {#each currentGroups as group (group.issuer + ':' + group.group_key)}
              <code class="bg-muted rounded px-1.5 py-0.5"
                >{group.group_name || group.group_key}</code
              >
            {/each}
          {:else}
            <span class="text-muted-foreground">No synchronized groups</span>
          {/if}
        </div>
      </div>
    </div>
  </div>

  <div class="border-border bg-card space-y-3 rounded-lg border p-4">
    <div class="flex items-center gap-2">
      <LockKeyhole class="text-muted-foreground size-4" />
      <h4 class="text-sm font-semibold">Approval boundary</h4>
    </div>
    <p class="text-muted-foreground text-xs">
      {approvalPolicies?.summary ??
        'RBAC decides whether a user can start an action. Approval policy stays reserved for future second-factor or approver requirements.'}
    </p>
    <div class="grid gap-3 text-xs sm:grid-cols-2">
      <div>
        <div class="text-muted-foreground">Policy status</div>
        <div class="mt-1 font-medium uppercase">{approvalPolicies?.status ?? 'reserved'}</div>
      </div>
      <div>
        <div class="text-muted-foreground">Stored rules</div>
        <div class="mt-1 font-medium">{approvalPolicies?.rules_count ?? 0}</div>
      </div>
    </div>
    <div class="text-muted-foreground text-xs">
      Audit attribution stays on the human principal, including project conversation confirms.
    </div>
  </div>
</div>

<div class="grid gap-4 xl:grid-cols-3">
  <SecuritySettingsHumanAuthAccessCard
    title="Instance effective access"
    subtitle="Control plane"
    roles={instancePermissions?.roles ?? []}
    permissions={instancePermissions?.permissions ?? []}
    emptyRoles="No instance roles"
    emptyPermissions="No instance permissions"
  />

  <SecuritySettingsHumanAuthAccessCard
    title="Organization effective access"
    subtitle={currentOrgName}
    roles={orgPermissions?.roles ?? []}
    permissions={orgPermissions?.permissions ?? []}
    emptyRoles="No organization roles"
    emptyPermissions="No organization permissions"
  />

  <SecuritySettingsHumanAuthAccessCard
    title="Project effective access"
    subtitle={currentProjectName}
    roles={projectPermissions?.roles ?? []}
    permissions={projectPermissions?.permissions ?? []}
    emptyRoles="No project roles"
    emptyPermissions="No project permissions"
  />
</div>

<div class="text-muted-foreground rounded-lg border border-dashed px-4 py-3 text-xs">
  Human permissions are evaluated by backend RBAC on resource/action keys. Agent scopes are related
  runtime token capabilities, but they are not reused as human permissions.
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
