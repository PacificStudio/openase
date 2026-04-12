<script lang="ts">
  import type { OIDCFormState } from '$lib/features/auth'
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Separator } from '$ui/separator'
  import { RefreshCcw, Rocket, Save, TestTube2 } from '@lucide/svelte'
  import { adminAuthT } from './i18n'
  import AdminAuthAdvancedDialog from './admin-auth-advanced-dialog.svelte'

  let {
    auth,
    form = $bindable({
      issuerURL: '',
      clientID: '',
      clientSecret: '',
      redirectMode: 'auto',
      fixedRedirectURL: '',
      scopesText: '',
      allowedDomainsText: '',
      bootstrapAdminEmailsText: '',
      sessionTTL: '',
      sessionIdleTTL: '',
    } satisfies OIDCFormState),
    actionKey = '',
    onSave,
    onTest,
    onEnable,
    onDisable,
  }: {
    auth: SecurityAuthSettings
    form?: OIDCFormState
    actionKey?: string
    onSave: () => void
    onTest: () => void
    onEnable: () => void
    onDisable: () => void
  } = $props()

  const scopesSummary = $derived(() => {
    const scopes = form.scopesText
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean)
    return scopes.length > 0 ? scopes.join(', ') : 'openid, profile, email'
  })

  const domainsSummary = $derived(() => {
    const domains = form.allowedDomainsText
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean)
    return domains.length > 0 ? domains.join(', ') : adminAuthT('adminAuth.summary.anyDomain')
  })

  const bootstrapSummary = $derived(() => {
    const emails = form.bootstrapAdminEmailsText
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean)
    return emails.length > 0 ? `${emails.length} email(s)` : adminAuthT('adminAuth.summary.none')
  })
</script>

<div class="border-border bg-card rounded-2xl border">
  <div class="p-6">
    <h3 class="text-base font-semibold">
      {adminAuthT('adminAuth.form.heading')}
    </h3>

    <!-- Primary fields: Provider connection -->
    <div class="mt-6 grid gap-4 sm:grid-cols-2">
      <div class="space-y-2">
        <Label for="admin-oidc-issuer-url">
          {adminAuthT('adminAuth.form.labels.issuerUrl')}
        </Label>
        <Input
          id="admin-oidc-issuer-url"
          value={form.issuerURL}
          placeholder={adminAuthT('adminAuth.form.placeholders.issuerUrl')}
          oninput={(event) => (form.issuerURL = (event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="admin-oidc-client-id">
          {adminAuthT('adminAuth.form.labels.clientId')}
        </Label>
        <Input
          id="admin-oidc-client-id"
          value={form.clientID}
          placeholder={adminAuthT('adminAuth.form.placeholders.clientId')}
          oninput={(event) => (form.clientID = (event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2 sm:col-span-2">
        <Label for="admin-oidc-client-secret">
          {adminAuthT('adminAuth.form.labels.clientSecret')}
        </Label>
        <Input
          id="admin-oidc-client-secret"
          type="password"
          value={form.clientSecret}
          placeholder={auth.oidc_draft.client_secret_configured
            ? adminAuthT('adminAuth.form.placeholders.clientSecretConfigured')
            : adminAuthT('adminAuth.form.placeholders.clientSecret')}
          oninput={(event) => (form.clientSecret = (event.currentTarget as HTMLInputElement).value)}
        />
        {#if auth.oidc_draft.client_secret_configured}
          <p class="text-muted-foreground text-[11px]">
            {adminAuthT('adminAuth.form.helper.clientSecretConfigured')}
          </p>
        {/if}
      </div>
    </div>

    <!-- Advanced settings summary + edit trigger -->
    <div class="mt-5">
      <Separator />
      <div class="mt-4 flex items-center justify-between gap-4">
        <div class="min-w-0 flex-1">
          <div class="text-muted-foreground text-xs font-medium">
            {adminAuthT('adminAuth.form.summaryLabel')}
          </div>
          <div class="text-muted-foreground mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-xs">
            <span>
              {adminAuthT('adminAuth.summary.scopes')}:{' '}
              <span class="text-foreground">{scopesSummary()}</span>
            </span>
            <span>
              {adminAuthT('adminAuth.summary.domains')}:{' '}
              <span class="text-foreground">{domainsSummary()}</span>
            </span>
            <span>
              {adminAuthT('adminAuth.summary.bootstrapAdmins')}:{' '}
              <span class="text-foreground">{bootstrapSummary()}</span>
            </span>
            <span>
              {adminAuthT('adminAuth.summary.sessionTtl')}:{' '}
              <span class="text-foreground">{form.sessionTTL || '0s'}</span>
            </span>
            <span>
              {adminAuthT('adminAuth.summary.idleTtl')}:{' '}
              <span class="text-foreground">{form.sessionIdleTTL || '0s'}</span>
            </span>
          </div>
        </div>
        <AdminAuthAdvancedDialog {form} />
      </div>
    </div>
  </div>

  <!-- Action footer -->
  <div class="flex flex-wrap items-center justify-between gap-3 border-t px-6 py-4">
    <Button
      variant="ghost"
      onclick={onDisable}
      disabled={actionKey !== ''}
      class="text-muted-foreground"
    >
      <RefreshCcw class="size-4" />
      {actionKey === 'disable'
        ? adminAuthT('adminAuth.actions.reverting')
        : adminAuthT('adminAuth.actions.revert')}
    </Button>
    <div class="flex flex-wrap gap-2">
      <Button variant="outline" onclick={onSave} disabled={actionKey !== ''}>
        <Save class="size-4" />
        {actionKey === 'save'
          ? adminAuthT('adminAuth.actions.savingDraft')
          : adminAuthT('adminAuth.actions.saveDraft')}
      </Button>
      <Button variant="outline" onclick={onTest} disabled={actionKey !== ''}>
        <TestTube2 class="size-4" />
        {actionKey === 'test'
          ? adminAuthT('adminAuth.actions.validating')
          : adminAuthT('adminAuth.actions.validate')}
      </Button>
      <Button onclick={onEnable} disabled={actionKey !== ''}>
        <Rocket class="size-4" />
        {actionKey === 'enable'
          ? adminAuthT('adminAuth.actions.activating')
          : adminAuthT('adminAuth.actions.activate')}
      </Button>
    </div>
  </div>
</div>
