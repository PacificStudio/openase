<script lang="ts">
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'

  let {
    value,
    onValueChange,
  }: {
    value: string
    onValueChange?: (value: string) => void
  } = $props()
</script>

<div class="space-y-2">
  <Label for="provider-auth-config">Plain auth config</Label>
  <Textarea
    id="provider-auth-config"
    rows={4}
    placeholder={`{\n  "base_url": "http://localhost:4318"\n}`}
    {value}
    oninput={(event) => onValueChange?.((event.currentTarget as HTMLTextAreaElement).value)}
  />
  <p class="text-muted-foreground text-xs">
    JSON object for non-secret provider settings only. Leave blank to clear.
  </p>
</div>
