<script lang="ts">
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { AlertTriangle, CheckCircle2, Rocket, Save, TestTube2 } from '@lucide/svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import OIDCRedirectFields from './oidc-redirect-fields.svelte'

  type OIDCFormState = {
    issuerURL: string
    clientID: string
    clientSecret: string
    redirectMode: 'auto' | 'fixed'
    fixedRedirectURL: string
    scopesText: string
    allowedDomainsText: string
    bootstrapAdminEmailsText: string
  }

  type OIDCTestResult = {
    status: string
    message: string
    issuer_url: string
    authorization_endpoint: string
    token_endpoint: string
    redirect_url: string
    warnings: string[]
  }

  type OIDCEnableResult = {
    status: string
    message: string
    restart_required: boolean
    next_steps: string[]
  }

  let {
    auth,
    form,
    actionKey = '',
    error = '',
    testResult = null,
    enableResult = null,
    onIssuerURL,
    onClientID,
    onClientSecret,
    onRedirectMode,
    onFixedRedirectURL,
    onScopes,
    onAllowedDomains,
    onBootstrapAdmins,
    onSave,
    onTest,
    onEnable,
  }: {
    auth: SecurityAuthSettings
    form: OIDCFormState
    actionKey?: string
    error?: string
    testResult?: OIDCTestResult | null
    enableResult?: OIDCEnableResult | null
    onIssuerURL: (value: string) => void
    onClientID: (value: string) => void
    onClientSecret: (value: string) => void
    onRedirectMode: (value: 'auto' | 'fixed') => void
    onFixedRedirectURL: (value: string) => void
    onScopes: (value: string) => void
    onAllowedDomains: (value: string) => void
    onBootstrapAdmins: (value: string) => void
    onSave: () => void
    onTest: () => void
    onEnable: () => void
  } = $props()
</script>

<div class="space-y-4">
  <div class="border-border bg-card space-y-4 rounded-lg border p-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <h4 class="text-sm font-semibold">Auth setup</h4>
          <Badge variant="outline">{auth.active_mode}</Badge>
          <Badge variant={auth.public_exposure_risk === 'high' ? 'destructive' : 'secondary'}>
            {auth.public_exposure_risk === 'high' ? 'High risk' : 'Local ready'}
          </Badge>
        </div>
        <p class="text-muted-foreground text-sm leading-relaxed">{auth.mode_summary}</p>
        <p class="text-muted-foreground text-xs leading-relaxed">
          Your current local admin principal is <code>{auth.local_principal}</code>. You can keep
          using local bootstrap links for personal or recovery access, or configure OIDC now and
          enable it only when you want managed multi-user browser access.
        </p>
      </div>

      <div class="grid gap-2 text-xs sm:min-w-64 sm:grid-cols-2 lg:w-80">
        <div>
          <div class="text-muted-foreground">Configured mode</div>
          <div class="mt-1 font-medium uppercase">{auth.configured_mode}</div>
        </div>
        <div>
          <div class="text-muted-foreground">Bootstrap admins</div>
          <div class="mt-1 font-medium">{auth.bootstrap_state.summary}</div>
        </div>
        <div class="sm:col-span-2">
          <div class="text-muted-foreground">Stored in</div>
          <div class="mt-1 font-mono break-all">{auth.config_path || 'Not available'}</div>
        </div>
      </div>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      {#each auth.warnings as warning (warning)}
        <div
          class={`rounded-lg border px-3 py-2 text-xs leading-relaxed ${
            auth.public_exposure_risk === 'high'
              ? 'border-amber-300 bg-amber-50 text-amber-900'
              : 'border-sky-200 bg-sky-50 text-sky-900'
          }`}
        >
          <div class="flex items-start gap-2">
            <AlertTriangle class="mt-0.5 size-4 shrink-0" />
            <span>{warning}</span>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <div class="border-border bg-card space-y-4 rounded-lg border p-4">
    <div>
      <h4 class="text-sm font-semibold">Draft OIDC configuration</h4>
      <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
        Save stores the draft for this instance without changing the active auth mode. Test checks
        provider discovery. Enable OIDC updates the configured auth mode and then tells you the next
        rollout step.
      </p>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="space-y-2">
        <Label for="oidc-issuer-url">Issuer URL</Label>
        <Input
          id="oidc-issuer-url"
          value={form.issuerURL}
          placeholder="https://idp.example.com/realms/openase"
          oninput={(event) => onIssuerURL((event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-client-id">Client ID</Label>
        <Input
          id="oidc-client-id"
          value={form.clientID}
          placeholder="openase"
          oninput={(event) => onClientID((event.currentTarget as HTMLInputElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-client-secret">Client secret</Label>
        <Input
          id="oidc-client-secret"
          type="password"
          value={form.clientSecret}
          placeholder={auth.oidc_draft.client_secret_configured
            ? 'Leave blank to keep the saved secret'
            : 'Paste the current client secret'}
          oninput={(event) => onClientSecret((event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">
          {auth.oidc_draft.client_secret_configured
            ? 'A client secret is already saved. Leave this field empty to preserve it.'
            : 'The client secret is stored server-side and never shown back in the UI.'}
        </p>
      </div>
      <OIDCRedirectFields
        redirectMode={form.redirectMode}
        fixedRedirectURL={form.fixedRedirectURL}
        {onRedirectMode}
        {onFixedRedirectURL}
      />
      <div class="space-y-2 lg:col-span-2">
        <Label for="oidc-scopes">Scopes</Label>
        <Textarea
          id="oidc-scopes"
          rows={3}
          value={form.scopesText}
          placeholder="openid, profile, email, groups"
          oninput={(event) => onScopes((event.currentTarget as HTMLTextAreaElement).value)}
        />
        <p class="text-muted-foreground text-[11px]">Use commas or new lines.</p>
      </div>
      <div class="space-y-2">
        <Label for="oidc-allowed-domains">Allowed domains</Label>
        <Textarea
          id="oidc-allowed-domains"
          rows={3}
          value={form.allowedDomainsText}
          placeholder="example.com"
          oninput={(event) => onAllowedDomains((event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>
      <div class="space-y-2">
        <Label for="oidc-bootstrap-admins">Bootstrap admin emails</Label>
        <Textarea
          id="oidc-bootstrap-admins"
          rows={3}
          value={form.bootstrapAdminEmailsText}
          placeholder="admin@example.com"
          oninput={(event) => onBootstrapAdmins((event.currentTarget as HTMLTextAreaElement).value)}
        />
      </div>
    </div>

    <div class="flex flex-wrap gap-3">
      <Button variant="outline" onclick={onSave} disabled={actionKey !== ''}>
        <Save class="size-4" />
        {actionKey === 'save' ? 'Saving…' : 'Save draft'}
      </Button>
      <Button variant="outline" onclick={onTest} disabled={actionKey !== ''}>
        <TestTube2 class="size-4" />
        {actionKey === 'test' ? 'Testing…' : 'Test configuration'}
      </Button>
      <Button onclick={onEnable} disabled={actionKey !== ''}>
        <Rocket class="size-4" />
        {actionKey === 'enable' ? 'Enabling…' : 'Enable OIDC'}
      </Button>
    </div>

    {#if error}
      <div class="text-destructive rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm">
        {error}
      </div>
    {/if}

    {#if testResult}
      <div
        class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-950"
      >
        <div class="flex items-start gap-2">
          <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
          <div class="space-y-2">
            <div class="font-medium">{testResult.message}</div>
            <div class="grid gap-2 text-xs md:grid-cols-2">
              <div>
                <div class="text-emerald-800/80">Issuer</div>
                <div class="font-mono break-all">{testResult.issuer_url}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">Redirect</div>
                <div class="font-mono break-all">{testResult.redirect_url}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">Authorization endpoint</div>
                <div class="font-mono break-all">{testResult.authorization_endpoint}</div>
              </div>
              <div>
                <div class="text-emerald-800/80">Token endpoint</div>
                <div class="font-mono break-all">{testResult.token_endpoint}</div>
              </div>
            </div>
            {#if testResult.warnings.length > 0}
              <div class="space-y-1 text-xs">
                {#each testResult.warnings as warning (warning)}
                  <div>{warning}</div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    {#if enableResult}
      <div
        class="rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm text-indigo-950"
      >
        <div class="flex items-start gap-2">
          <CheckCircle2 class="mt-0.5 size-4 shrink-0" />
          <div class="space-y-2">
            <div class="font-medium">{enableResult.message}</div>
            {#if enableResult.restart_required}
              <div class="text-xs font-medium tracking-wide text-indigo-700 uppercase">
                Restart required
              </div>
            {/if}
            <ol class="list-inside list-decimal space-y-1 text-xs leading-relaxed">
              {#each enableResult.next_steps as step (step)}
                <li>{step}</li>
              {/each}
            </ol>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
