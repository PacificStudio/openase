<script lang="ts">
  import * as Select from '$ui/select'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

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
  <Label for={`${idPrefix}-redirect-mode`}>Redirect mode</Label>
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
      {redirectMode === 'fixed' ? 'Fixed redirect URL' : 'Auto-derived from current request'}
    </Select.Trigger>
    <Select.Content>
      <Select.Item value="auto">Auto-derived from current request</Select.Item>
      <Select.Item value="fixed">Fixed redirect URL</Select.Item>
    </Select.Content>
  </Select.Root>
  <p class="text-muted-foreground text-[11px]">
    Auto derives <code>/api/v1/auth/oidc/callback</code> from the current public scheme and host, which
    is the safest option for desktop ports, proxies, and multiple environments.
  </p>
</div>

<div class="space-y-2">
  <Label for={`${idPrefix}-fixed-redirect-url`}>Fixed redirect URL</Label>
  <Input
    id={`${idPrefix}-fixed-redirect-url`}
    value={fixedRedirectURL}
    placeholder="https://openase.example.com/api/v1/auth/oidc/callback"
    disabled={redirectMode !== 'fixed'}
    oninput={(event) => onFixedRedirectURL((event.currentTarget as HTMLInputElement).value)}
  />
  <p class="text-muted-foreground text-[11px]">
    {redirectMode === 'fixed'
      ? 'Use this only when the identity provider requires one exact registered callback URL.'
      : 'Optional saved override for strict IdPs. Auto mode ignores this field until you switch back to fixed.'}
  </p>
</div>
