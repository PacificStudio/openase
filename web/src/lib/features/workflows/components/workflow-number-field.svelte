<script lang="ts">
  import type { Component } from 'svelte'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

  let {
    id,
    label,
    value,
    icon,
    disabled = false,
    oninput,
  }: {
    id: string
    label: string
    value: string
    icon: Component
    disabled?: boolean
    oninput?: (value: string) => void
  } = $props()

  const Icon = $derived(icon)
</script>

<div class="space-y-2">
  <Label for={id}>{label}</Label>
  <div class="relative">
    <Icon
      class="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
    />
    <Input
      {id}
      class="pl-9"
      inputmode="numeric"
      {value}
      {disabled}
      oninput={(event) => oninput?.((event.currentTarget as HTMLInputElement).value)}
    />
  </div>
</div>
