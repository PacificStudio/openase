<script lang="ts">
  import { oidcSessionFieldCopy, type OIDCFormState } from '$lib/features/auth'
  import type { SecurityAuthSettings } from '$lib/api/contracts'
  import * as Dialog from '$ui/dialog'
  import * as Select from '$ui/select'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { Separator } from '$ui/separator'
  import { RefreshCcw, Rocket, Save, Settings2, TestTube2 } from '@lucide/svelte'

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

  let advancedOpen = $state(false)

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
    return domains.length > 0 ? domains.join(', ') : 'Any domain'
  })

  const bootstrapSummary = $derived(() => {
    const emails = form.bootstrapAdminEmailsText
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean)
    return emails.length > 0 ? `${emails.length} email(s)` : 'None'
  })
</script>

<div class="border-border bg-card rounded-2xl border">
  <div class="p-6">
    <h3 class="text-base font-semibold">OIDC Configuration</h3>

    <!-- Primary fields: Provider connection -->
    <div class="mt-6 grid gap-4 sm:grid-cols-2">
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
      <div class="space-y-2 sm:col-span-2">
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
        {#if auth.oidc_draft.client_secret_configured}
          <p class="text-muted-foreground text-[11px]">Already configured. Leave blank to keep.</p>
        {/if}
      </div>
    </div>

    <!-- Advanced settings summary + edit trigger -->
    <div class="mt-5">
      <Separator />
      <div class="mt-4 flex items-center justify-between gap-4">
        <div class="min-w-0 flex-1">
          <div class="text-muted-foreground text-xs font-medium">Advanced settings</div>
          <div class="text-muted-foreground mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-xs">
            <span>Scopes: <span class="text-foreground">{scopesSummary()}</span></span>
            <span>Domains: <span class="text-foreground">{domainsSummary()}</span></span>
            <span>Bootstrap admins: <span class="text-foreground">{bootstrapSummary()}</span></span>
            <span>Session TTL: <span class="text-foreground">{form.sessionTTL || '0s'}</span></span>
            <span>Idle TTL: <span class="text-foreground">{form.sessionIdleTTL || '0s'}</span></span
            >
          </div>
        </div>
        <Dialog.Root bind:open={advancedOpen}>
          <Dialog.Trigger>
            {#snippet child({ props })}
              <Button variant="outline" size="sm" {...props}>
                <Settings2 class="size-4" />
                Edit
              </Button>
            {/snippet}
          </Dialog.Trigger>
          <Dialog.Content class="max-w-lg">
            <Dialog.Header>
              <Dialog.Title>Advanced Settings</Dialog.Title>
              <Dialog.Description>
                Redirect, scopes, session policy, and access policy.
              </Dialog.Description>
            </Dialog.Header>
            <Dialog.Body>
              <div class="space-y-5">
                <!-- Redirect mode -->
                <div class="space-y-2">
                  <Label for="admin-oidc-redirect-mode">Redirect mode</Label>
                  <Select.Root
                    type="single"
                    value={form.redirectMode}
                    onValueChange={(value) => {
                      if (value === 'auto' || value === 'fixed') {
                        form.redirectMode = value
                      }
                    }}
                  >
                    <Select.Trigger id="admin-oidc-redirect-mode" class="h-10 w-full text-sm">
                      {form.redirectMode === 'fixed'
                        ? 'Fixed redirect URL'
                        : 'Auto-derived from current request'}
                    </Select.Trigger>
                    <Select.Content>
                      <Select.Item value="auto">Auto-derived from current request</Select.Item>
                      <Select.Item value="fixed">Fixed redirect URL</Select.Item>
                    </Select.Content>
                  </Select.Root>
                  <p class="text-muted-foreground text-[11px]">
                    Auto derives callback URL from current host.
                  </p>
                </div>

                {#if form.redirectMode === 'fixed'}
                  <div class="space-y-2">
                    <Label for="admin-oidc-fixed-redirect-url">Fixed redirect URL</Label>
                    <Input
                      id="admin-oidc-fixed-redirect-url"
                      value={form.fixedRedirectURL}
                      placeholder="https://openase.example.com/api/v1/auth/oidc/callback"
                      oninput={(event) =>
                        (form.fixedRedirectURL = (event.currentTarget as HTMLInputElement).value)}
                    />
                  </div>
                {/if}

                <Separator />

                <div class="space-y-2">
                  <Label for="admin-oidc-session-ttl">Session TTL</Label>
                  <Input
                    id="admin-oidc-session-ttl"
                    value={form.sessionTTL}
                    placeholder="8h"
                    oninput={(event) =>
                      (form.sessionTTL = (event.currentTarget as HTMLInputElement).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {oidcSessionFieldCopy.sessionTTLDescription}
                  </p>
                </div>

                <div class="space-y-2">
                  <Label for="admin-oidc-session-idle-ttl">Idle TTL</Label>
                  <Input
                    id="admin-oidc-session-idle-ttl"
                    value={form.sessionIdleTTL}
                    placeholder="30m"
                    oninput={(event) =>
                      (form.sessionIdleTTL = (event.currentTarget as HTMLInputElement).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {oidcSessionFieldCopy.sessionIdleTTLDescription}
                  </p>
                </div>

                <Separator />

                <!-- Scopes -->
                <div class="space-y-2">
                  <Label for="admin-oidc-scopes">Scopes</Label>
                  <Textarea
                    id="admin-oidc-scopes"
                    rows={2}
                    value={form.scopesText}
                    placeholder="openid, profile, email, groups"
                    oninput={(event) =>
                      (form.scopesText = (event.currentTarget as HTMLTextAreaElement).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">Comma or newline separated.</p>
                </div>

                <Separator />

                <!-- Allowed domains -->
                <div class="space-y-2">
                  <Label for="admin-oidc-allowed-domains">Allowed domains</Label>
                  <Textarea
                    id="admin-oidc-allowed-domains"
                    rows={2}
                    value={form.allowedDomainsText}
                    placeholder="example.com"
                    oninput={(event) =>
                      (form.allowedDomainsText = (
                        event.currentTarget as HTMLTextAreaElement
                      ).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">Leave blank to allow all.</p>
                </div>

                <!-- Bootstrap admin emails -->
                <div class="space-y-2">
                  <Label for="admin-oidc-bootstrap-admins">Bootstrap admin emails</Label>
                  <Textarea
                    id="admin-oidc-bootstrap-admins"
                    rows={2}
                    value={form.bootstrapAdminEmailsText}
                    placeholder="admin@example.com"
                    oninput={(event) =>
                      (form.bootstrapAdminEmailsText = (
                        event.currentTarget as HTMLTextAreaElement
                      ).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    Auto-granted admin on first login.
                  </p>
                </div>
              </div>
            </Dialog.Body>
            <Dialog.Footer>
              <Dialog.Close>
                {#snippet child({ props })}
                  <Button variant="outline" {...props}>Done</Button>
                {/snippet}
              </Dialog.Close>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Root>
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
      {actionKey === 'disable' ? 'Switching…' : 'Revert to local'}
    </Button>
    <div class="flex flex-wrap gap-2">
      <Button variant="outline" onclick={onSave} disabled={actionKey !== ''}>
        <Save class="size-4" />
        {actionKey === 'save' ? 'Saving…' : 'Save draft'}
      </Button>
      <Button variant="outline" onclick={onTest} disabled={actionKey !== ''}>
        <TestTube2 class="size-4" />
        {actionKey === 'test' ? 'Validating…' : 'Validate'}
      </Button>
      <Button onclick={onEnable} disabled={actionKey !== ''}>
        <Rocket class="size-4" />
        {actionKey === 'enable' ? 'Activating…' : 'Activate OIDC'}
      </Button>
    </div>
  </div>
</div>
