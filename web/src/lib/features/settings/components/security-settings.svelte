<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ProjectSecurity } from '$lib/api/contracts'
  import { getProjectSecurity } from '$lib/api/openase'
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import * as Card from '$ui/card'
  import { Separator } from '$ui/separator'

  const securityCapability = capabilityCatalog.securitySettings

  let security = $state<ProjectSecurity | null>(null)
  let loading = $state(false)
  let error = $state('')

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      security = null
      error = ''
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''
      security = null

      try {
        const payload = await getProjectSecurity(projectId)
        if (cancelled) return
        security = payload.security
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load security settings.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function surfaceTone(exposed: boolean, configured: boolean) {
    if (exposed && configured) {
      return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
    }
    if (exposed) {
      return 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300'
    }
    return 'border-slate-500/40 bg-slate-500/10 text-slate-700 dark:text-slate-300'
  }

  function surfaceLabel(exposed: boolean, configured: boolean) {
    if (exposed && configured) return 'Guarded'
    if (exposed) return 'Needs Secret'
    return 'Not Exposed'
  }

  function formatTimestamp(value: string | null | undefined) {
    if (!value) return 'Never'
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) return value
    return parsed.toLocaleString()
  }
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Security</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(securityCapability.state)}`}
      >
        {capabilityStateLabel(securityCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 text-sm">{securityCapability.summary}</p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading security settings…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else if security}
    <div class="grid gap-4 xl:grid-cols-2">
      {#each security.surfaces as surface (surface.key)}
        <Card.Root>
          <Card.Header>
            <div class="flex items-center justify-between gap-3">
              <Card.Title>{surface.label}</Card.Title>
              <span
                class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${surfaceTone(surface.exposed, surface.configured)}`}
              >
                {surfaceLabel(surface.exposed, surface.configured)}
              </span>
            </div>
            <Card.Description>{surface.summary}</Card.Description>
          </Card.Header>
        </Card.Root>
      {/each}
    </div>

    <Card.Root>
      <Card.Header>
        <Card.Title>Agent token exposure</Card.Title>
        <Card.Description>{security.agent_platform.summary}</Card.Description>
      </Card.Header>
      <Card.Content class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <div class="border-border rounded-md border px-3 py-3">
            <div class="text-muted-foreground text-xs uppercase">Runtime</div>
            <div class="text-foreground mt-1 text-sm font-medium">
              {security.agent_platform.exposed ? 'Exposed' : 'Hidden'}
            </div>
          </div>
          <div class="border-border rounded-md border px-3 py-3">
            <div class="text-muted-foreground text-xs uppercase">Active tokens</div>
            <div class="text-foreground mt-1 text-sm font-medium">
              {security.agent_platform.active_token_count}
            </div>
          </div>
          <div class="border-border rounded-md border px-3 py-3">
            <div class="text-muted-foreground text-xs uppercase">Expired tokens</div>
            <div class="text-foreground mt-1 text-sm font-medium">
              {security.agent_platform.expired_token_count}
            </div>
          </div>
          <div class="border-border rounded-md border px-3 py-3">
            <div class="text-muted-foreground text-xs uppercase">Runtime mode</div>
            <div class="text-foreground mt-1 text-sm font-medium">{security.runtime_mode}</div>
          </div>
        </div>

        <div class="grid gap-4 xl:grid-cols-2">
          <div class="border-border rounded-md border px-4 py-4">
            <div class="text-foreground text-sm font-medium">Default token scopes</div>
            <div class="text-muted-foreground mt-1 text-xs">
              Automatically granted when a token is issued without an explicit scope list.
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              {#each security.agent_platform.default_scopes as scope (scope)}
                <span class="bg-muted text-foreground rounded-full px-2 py-1 text-xs">{scope}</span>
              {/each}
            </div>
          </div>

          <div class="border-border rounded-md border px-4 py-4">
            <div class="text-foreground text-sm font-medium">Privileged mutation scopes</div>
            <div class="text-muted-foreground mt-1 text-xs">
              Scopes that expand tokens from ticket work into direct project mutation APIs.
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              {#each security.agent_platform.privileged_scopes as scope (scope)}
                <span class="bg-muted text-foreground rounded-full px-2 py-1 text-xs">{scope}</span>
              {/each}
            </div>
          </div>
        </div>

        <div class="grid gap-4 xl:grid-cols-2">
          <div class="border-border rounded-md border px-4 py-4">
            <div class="text-foreground text-sm font-medium">Last token issued</div>
            <div class="text-muted-foreground mt-2 text-sm">
              {formatTimestamp(security.agent_platform.last_token_issued_at)}
            </div>
          </div>
          <div class="border-border rounded-md border px-4 py-4">
            <div class="text-foreground text-sm font-medium">Last token used</div>
            <div class="text-muted-foreground mt-2 text-sm">
              {formatTimestamp(security.agent_platform.last_token_used_at)}
            </div>
          </div>
        </div>
      </Card.Content>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Boundary notes</Card.Title>
        <Card.Description>
          This section is intentionally audit-first. It describes what the current app surface exposes
          instead of implying that a full auth console already exists.
        </Card.Description>
      </Card.Header>
      <Card.Content>
        <ul class="text-muted-foreground space-y-2 text-sm">
          {#each security.notes as note (note)}
            <li>{note}</li>
          {/each}
        </ul>
      </Card.Content>
    </Card.Root>
  {/if}
</div>
