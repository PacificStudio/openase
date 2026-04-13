<script lang="ts">
  import { Settings2 } from '@lucide/svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Separator } from '$ui/separator'
  import { Textarea } from '$ui/textarea'
  import { oidcSessionFieldCopy, type OIDCFormState } from '$lib/features/auth'
  import { adminAuthT } from './i18n'

  let { form }: { form: OIDCFormState } = $props()
</script>

<Dialog.Root>
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
      <Dialog.Title>{adminAuthT('adminAuth.dialog.title')}</Dialog.Title>
      <Dialog.Description>{adminAuthT('adminAuth.dialog.description')}</Dialog.Description>
    </Dialog.Header>
    <Dialog.Body>
      <div class="space-y-5">
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
              <Select.Item value="auto">{adminAuthT('adminAuth.options.redirectAuto')}</Select.Item>
              <Select.Item value="fixed"
                >{adminAuthT('adminAuth.options.redirectFixed')}</Select.Item
              >
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
            oninput={(event) => (form.sessionTTL = (event.currentTarget as HTMLInputElement).value)}
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
              (form.allowedDomainsText = (event.currentTarget as HTMLTextAreaElement).value)}
          />
          <p class="text-muted-foreground text-[11px]">
            {adminAuthT('adminAuth.helper.allowedDomains')}
          </p>
        </div>

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
              (form.bootstrapAdminEmailsText = (event.currentTarget as HTMLTextAreaElement).value)}
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
