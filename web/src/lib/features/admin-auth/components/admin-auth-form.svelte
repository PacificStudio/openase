<script lang="ts">
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { RefreshCcw, Rocket, Save, TestTube2 } from '@lucide/svelte'

  type OIDCFormState = {
    issuerURL: string
    clientID: string
    clientSecret: string
    redirectURL: string
    scopesText: string
    allowedDomainsText: string
    bootstrapAdminEmailsText: string
  }

  let {
    auth,
    form,
    actionKey = '',
    onSave,
    onTest,
    onEnable,
    onDisable,
  }: {
    auth: SecurityAuthSettings
    form: OIDCFormState
    actionKey?: string
    onSave: () => void
    onTest: () => void
    onEnable: () => void
    onDisable: () => void
  } = $props()
</script>

<div class="border-border bg-card space-y-5 rounded-2xl border p-5">
  <div class="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
    <div class="text-sm font-semibold">OIDC configuration</div>

    <div class="flex flex-wrap gap-2">
      <Button variant="outline" onclick={onSave} disabled={actionKey !== ''}>
        <Save class="size-4" />
        {actionKey === 'save' ? 'Saving…' : 'Save configuration'}
      </Button>
      <Button variant="outline" onclick={onTest} disabled={actionKey !== ''}>
        <TestTube2 class="size-4" />
        {actionKey === 'test' ? 'Testing…' : 'Test configuration'}
      </Button>
      <Button onclick={onEnable} disabled={actionKey !== ''}>
        <Rocket class="size-4" />
        {actionKey === 'enable' ? 'Enabling…' : 'Enable OIDC'}
      </Button>
      <Button variant="outline" onclick={onDisable} disabled={actionKey !== ''}>
        <RefreshCcw class="size-4" />
        {actionKey === 'disable' ? 'Reverting…' : 'Revert to disabled'}
      </Button>
    </div>
  </div>

  <div class="grid gap-4 lg:grid-cols-2">
    <div class="space-y-2">
      <Label for="admin-oidc-issuer-url">Issuer URL</Label>
      <Input
        id="admin-oidc-issuer-url"
        value={form.issuerURL}
        placeholder="https://idp.example.com/realms/openase"
        oninput={(event) => (form.issuerURL = (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="admin-oidc-client-id">Client ID</Label>
      <Input
        id="admin-oidc-client-id"
        value={form.clientID}
        placeholder="openase"
        oninput={(event) => (form.clientID = (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="admin-oidc-client-secret">Client secret</Label>
      <Input
        id="admin-oidc-client-secret"
        type="password"
        value={form.clientSecret}
        placeholder={auth.oidc_draft.client_secret_configured
          ? 'Leave blank to keep the saved secret'
          : 'Paste the current client secret'}
        oninput={(event) => (form.clientSecret = (event.currentTarget as HTMLInputElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">
        {auth.oidc_draft.client_secret_configured
          ? 'A client secret is already stored server-side. Leave this blank to preserve it.'
          : 'The client secret is accepted server-side and never echoed back in plain text.'}
      </p>
    </div>
    <div class="space-y-2">
      <Label for="admin-oidc-redirect-url">Redirect URL</Label>
      <Input
        id="admin-oidc-redirect-url"
        value={form.redirectURL}
        placeholder="http://127.0.0.1:19836/api/v1/auth/oidc/callback"
        oninput={(event) => (form.redirectURL = (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
    <div class="space-y-2 lg:col-span-2">
      <Label for="admin-oidc-scopes">Scopes</Label>
      <Textarea
        id="admin-oidc-scopes"
        rows={3}
        value={form.scopesText}
        placeholder="openid, profile, email, groups"
        oninput={(event) => (form.scopesText = (event.currentTarget as HTMLTextAreaElement).value)}
      />
      <p class="text-muted-foreground text-[11px]">Use commas or new lines.</p>
    </div>
    <div class="space-y-2">
      <Label for="admin-oidc-allowed-domains">Allowed domains</Label>
      <Textarea
        id="admin-oidc-allowed-domains"
        rows={3}
        value={form.allowedDomainsText}
        placeholder="example.com"
        oninput={(event) =>
          (form.allowedDomainsText = (event.currentTarget as HTMLTextAreaElement).value)}
      />
    </div>
    <div class="space-y-2">
      <Label for="admin-oidc-bootstrap-admins">Bootstrap admin emails</Label>
      <Textarea
        id="admin-oidc-bootstrap-admins"
        rows={3}
        value={form.bootstrapAdminEmailsText}
        placeholder="admin@example.com"
        oninput={(event) =>
          (form.bootstrapAdminEmailsText = (event.currentTarget as HTMLTextAreaElement).value)}
      />
    </div>
  </div>
</div>
