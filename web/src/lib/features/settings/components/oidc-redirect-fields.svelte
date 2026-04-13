<script lang="ts">
  import * as Select from '$ui/select'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    redirectMode,
    fixedRedirectURL,
    onRedirectMode,
    onFixedRedirectURL,
    idPrefix = 'oidc',
  }: {
    redirectMode: 'auto' | 'fixed'
    fixedRedirectURL: string
    onRedirectMode: (value: 'auto' | 'fixed') => void
    onFixedRedirectURL: (value: string) => void
    idPrefix?: string
  } = $props()
</script>

<div class="space-y-2">
  <Label for={`${idPrefix}-redirect-mode`}>
    {i18nStore.t('settings.oidcRedirect.labels.redirectMode')}
  </Label>
  <Select.Root
    type="single"
    value={redirectMode}
    onValueChange={(value) => {
      if (value === 'auto' || value === 'fixed') {
        onRedirectMode(value)
      }
    }}
  >
    <Select.Trigger id={`${idPrefix}-redirect-mode`} class="h-10 w-full text-sm">
      {redirectMode === 'fixed'
        ? i18nStore.t('settings.oidcRedirect.labels.fixed')
        : i18nStore.t('settings.oidcRedirect.labels.auto')}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="auto">
        {i18nStore.t('settings.oidcRedirect.labels.auto')}
      </Select.Item>
      <Select.Item value="fixed">
        {i18nStore.t('settings.oidcRedirect.labels.fixed')}
      </Select.Item>
    </Select.Content>
  </Select.Root>
  <p class="text-muted-foreground text-[11px]">
    {i18nStore.t('settings.oidcRedirect.hints.auto', {
      callback: '/api/v1/auth/oidc/callback',
    })}
  </p>
</div>

<div class="space-y-2">
  <Label for={`${idPrefix}-fixed-redirect-url`}>
    {i18nStore.t('settings.oidcRedirect.labels.fixedUrl')}
  </Label>
  <Input
    id={`${idPrefix}-fixed-redirect-url`}
    value={fixedRedirectURL}
    placeholder={i18nStore.t('settings.oidcRedirect.placeholders.fixedUrl')}
    disabled={redirectMode !== 'fixed'}
    oninput={(event) => onFixedRedirectURL((event.currentTarget as HTMLInputElement).value)}
  />
  <p class="text-muted-foreground text-[11px]">
    {redirectMode === 'fixed'
      ? i18nStore.t('settings.oidcRedirect.hints.fixed')
      : i18nStore.t('settings.oidcRedirect.hints.optional')}
  </p>
</div>
