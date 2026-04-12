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
  import { adminAuthT } from './i18n'

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
    return domains.length > 0
      ? domains.join(', ')
      : adminAuthT('adminAuth.summary.anyDomain')
  })

  const bootstrapSummary = $derived(() => {
    const emails = form.bootstrapAdminEmailsText
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter(Boolean)
    return emails.length > 0
      ? `${emails.length} email(s)`
      : adminAuthT('adminAuth.summary.none')
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
              <Dialog.Title>
                {adminAuthT('adminAuth.dialog.title')}
              </Dialog.Title>
              <Dialog.Description>
                {adminAuthT('adminAuth.dialog.description')}
              </Dialog.Description>
            </Dialog.Header>
            <Dialog.Body>
              <div class="space-y-5">
                <!-- Redirect mode -->
                <div class="space-y-2">
                  <Label for="admin-oidc-redirect-mode">
                    {adminAuthT('adminAuth.form.labels.redirectMode')}
                  </Label>
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
                        ? adminAuthT('adminAuth.options.redirectFixed')
                        : adminAuthT('adminAuth.options.redirectAuto')}
                    </Select.Trigger>
                    <Select.Content>
                      <Select.Item value="auto">
                        {adminAuthT('adminAuth.options.redirectAuto')}
                      </Select.Item>
                      <Select.Item value="fixed">
                        {adminAuthT('adminAuth.options.redirectFixed')}
                      </Select.Item>
                    </Select.Content>
                  </Select.Root>
                  <p class="text-muted-foreground text-[11px]">
                    {adminAuthT('adminAuth.helper.redirectMode')}
                  </p>
                </div>

                {#if form.redirectMode === 'fixed'}
                  <div class="space-y-2">
                  <Label for="admin-oidc-fixed-redirect-url">
                    {adminAuthT('adminAuth.form.labels.fixedRedirectUrl')}
                  </Label>
                  <Input
                    id="admin-oidc-fixed-redirect-url"
                    value={form.fixedRedirectURL}
                    placeholder={adminAuthT('adminAuth.form.placeholders.fixedRedirectUrl')}
                    oninput={(event) =>
                      (form.fixedRedirectURL = (event.currentTarget as HTMLInputElement).value)}
                  />
                  </div>
                {/if}

                <Separator />

                <div class="space-y-2">
                  <Label for="admin-oidc-session-ttl">
                    {adminAuthT('adminAuth.form.labels.sessionTtl')}
                  </Label>
                  <Input
                    id="admin-oidc-session-ttl"
                    value={form.sessionTTL}
                    placeholder={adminAuthT('adminAuth.form.placeholders.sessionTtl')}
                    oninput={(event) =>
                      (form.sessionTTL = (event.currentTarget as HTMLInputElement).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {oidcSessionFieldCopy.sessionTTLDescription}
                  </p>
                </div>

                <div class="space-y-2">
                  <Label for="admin-oidc-session-idle-ttl">
                    {adminAuthT('adminAuth.form.labels.idleTtl')}
                  </Label>
                  <Input
                    id="admin-oidc-session-idle-ttl"
                    value={form.sessionIdleTTL}
                    placeholder={adminAuthT('adminAuth.form.placeholders.idleTtl')}
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
                  <Label for="admin-oidc-scopes">
                    {adminAuthT('adminAuth.form.labels.scopes')}
                  </Label>
                  <Textarea
                    id="admin-oidc-scopes"
                    rows={2}
                    value={form.scopesText}
                    placeholder={adminAuthT('adminAuth.form.placeholders.scopes')}
                    oninput={(event) =>
                      (form.scopesText = (event.currentTarget as HTMLTextAreaElement).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {adminAuthT('adminAuth.helper.scopes')}
                  </p>
                </div>

                <Separator />

                <!-- Allowed domains -->
                <div class="space-y-2">
                  <Label for="admin-oidc-allowed-domains">
                    {adminAuthT('adminAuth.form.labels.domains')}
                  </Label>
                  <Textarea
                    id="admin-oidc-allowed-domains"
                    rows={2}
                    value={form.allowedDomainsText}
                    placeholder={adminAuthT('adminAuth.form.placeholders.allowedDomains')}
                    oninput={(event) =>
                      (form.allowedDomainsText = (
                        event.currentTarget as HTMLTextAreaElement
                      ).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {adminAuthT('adminAuth.helper.allowedDomains')}
                  </p>
                </div>

                <!-- Bootstrap admin emails -->
                <div class="space-y-2">
                  <Label for="admin-oidc-bootstrap-admins">
                    {adminAuthT('adminAuth.form.labels.bootstrapAdmins')}
                  </Label>
                  <Textarea
                    id="admin-oidc-bootstrap-admins"
                    rows={2}
                    value={form.bootstrapAdminEmailsText}
                    placeholder={adminAuthT('adminAuth.form.placeholders.bootstrapAdminEmails')}
                    oninput={(event) =>
                      (form.bootstrapAdminEmailsText = (
                        event.currentTarget as HTMLTextAreaElement
                      ).value)}
                  />
                  <p class="text-muted-foreground text-[11px]">
                    {adminAuthT('adminAuth.helper.bootstrapAdminEmails')}
                  </p>
                </div>
              </div>
            </Dialog.Body>
            <Dialog.Footer>
              <Dialog.Close>
                {#snippet child({ props })}
                  <Button variant="outline" {...props}>
                    {adminAuthT('adminAuth.dialog.actionDone')}
                  </Button>
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
