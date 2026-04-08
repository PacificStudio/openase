<script lang="ts">
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { authStore } from '$lib/stores/auth.svelte'
  import { Badge } from '$ui/badge'
  import { ArrowRight, KeyRound, ShieldCheck, Users } from '@lucide/svelte'
  import SecuritySettingsHumanAuthGuideLinks from './security-settings-human-auth-guide-links.svelte'

  let {
    auth = null,
    organizationId = '',
    projectAccessCurrent = false,
    projectAccessHref = '#access',
    showDocs = false,
  }: {
    auth?: SecurityAuthSettings | null
    organizationId?: string
    projectAccessCurrent?: boolean
    projectAccessHref?: string
    showDocs?: boolean
  } = $props()

  const activeMode = $derived(auth?.active_mode || authStore.authMode || 'disabled')
  const configuredMode = $derived(auth?.configured_mode || activeMode)
  const orgAdminBase = $derived(organizationId ? `${organizationPath(organizationId)}/admin` : '')
  const modeSummary = $derived(
    auth?.mode_summary ||
      (activeMode === 'oidc'
        ? 'OIDC is active, so real human roles now decide who can reach instance, organization, and project controls.'
        : 'Disabled mode keeps the local bootstrap principal active, but instance and organization IAM still live outside project security.'),
  )
  const recommendedMode = $derived(auth?.recommended_mode || '')
</script>

<div class="space-y-4">
  <div class="rounded-2xl border border-amber-200/80 bg-amber-50/80 p-4 text-sm text-amber-950">
    <div class="flex flex-wrap items-center gap-2">
      <Badge variant="outline">Migration note</Badge>
      <Badge variant="secondary">Active: {activeMode}</Badge>
      <Badge variant="outline">Configured: {configuredMode}</Badge>
    </div>
    <p class="mt-3 leading-6">{modeSummary}</p>
    {#if recommendedMode}
      <p class="mt-2 text-xs leading-5 opacity-90">{recommendedMode}</p>
    {/if}
  </div>

  <div class="grid gap-4 xl:grid-cols-3">
    <div class="rounded-2xl border bg-white p-4 shadow-sm">
      <div class="flex items-center gap-2">
        <ShieldCheck class="text-muted-foreground size-4" />
        <div class="text-sm font-semibold">Instance auth and directory</div>
      </div>
      <p class="text-muted-foreground mt-2 text-sm leading-6">
        Manage auth mode, OIDC rollout, bootstrap admins, user directory, and session governance
        under <code>/admin</code> instead of project settings.
      </p>
      <a
        class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-sky-700 hover:text-sky-800"
        href="/admin/auth"
      >
        Open <code>/admin/auth</code>
        <ArrowRight class="size-4" />
      </a>
    </div>

    <div class="rounded-2xl border bg-white p-4 shadow-sm">
      <div class="flex items-center gap-2">
        <Users class="text-muted-foreground size-4" />
        <div class="text-sm font-semibold">Org members, invites, and roles</div>
      </div>
      <p class="text-muted-foreground mt-2 text-sm leading-6">
        Org membership lifecycle and org-scoped RBAC now live in org admin so project settings no
        longer doubles as the people-governance surface.
      </p>
      {#if orgAdminBase}
        <a
          class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-sky-700 hover:text-sky-800"
          href={`${orgAdminBase}/members`}
        >
          Open org admin
          <ArrowRight class="size-4" />
        </a>
      {:else}
        <div class="text-muted-foreground mt-4 text-xs">
          Select an organization to open org admin.
        </div>
      {/if}
    </div>

    <div class="rounded-2xl border bg-white p-4 shadow-sm">
      <div class="flex items-center gap-2">
        <KeyRound class="text-muted-foreground size-4" />
        <div class="text-sm font-semibold">Project access stays here</div>
      </div>
      <p class="text-muted-foreground mt-2 text-sm leading-6">
        Keep project-scoped bindings and effective project access in Settings -&gt; Access. Project
        Security stays focused on outbound credentials, webhooks, and runtime token posture.
      </p>
      {#if projectAccessCurrent}
        <div class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-slate-700">
          Current surface
          <Badge variant="outline">Settings -&gt; Access</Badge>
        </div>
      {:else}
        <a
          class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-sky-700 hover:text-sky-800"
          href={projectAccessHref}
        >
          Open Settings -&gt; Access
          <ArrowRight class="size-4" />
        </a>
      {/if}
    </div>
  </div>

  {#if showDocs && auth?.docs?.length}
    <SecuritySettingsHumanAuthGuideLinks docs={auth.docs} />
  {/if}
</div>
