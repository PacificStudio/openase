<script lang="ts">
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { requiredProviderSecretBindingEnvVars } from '../provider-secret-bindings'

  let {
    adapterType,
    value,
    onValueChange,
  }: {
    adapterType: string
    value: string
    onValueChange?: (value: string) => void
  } = $props()

  const requiredSecretEnvVars = $derived(requiredProviderSecretBindingEnvVars(adapterType))
  const placeholder = $derived.by(() => {
    if (requiredSecretEnvVars.length === 0) {
      return `{\n  "ENV_VAR_NAME": "PROJECT_SECRET_ALIAS"\n}`
    }

    return JSON.stringify(
      Object.fromEntries(requiredSecretEnvVars.map((envVar) => [envVar, envVar])),
      null,
      2,
    )
  })
</script>

<div class="space-y-2">
  <Label for="provider-secret-bindings">Secret bindings</Label>
  <Textarea
    id="provider-secret-bindings"
    rows={5}
    {placeholder}
    {value}
    oninput={(event) => onValueChange?.((event.currentTarget as HTMLTextAreaElement).value)}
  />
  <p class="text-muted-foreground text-xs">
    JSON object mapping runtime env vars to OpenASE secret aliases. Leave blank to keep using legacy
    inline secrets during migration.
  </p>
  {#if requiredSecretEnvVars.length > 0}
    <p class="text-muted-foreground text-xs">
      Recommended env vars for this adapter: {requiredSecretEnvVars.join(', ')}.
    </p>
  {/if}
</div>
