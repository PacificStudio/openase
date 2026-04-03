<script lang="ts">
  import type { Component } from 'svelte'
  import { Input } from '$ui/input'

  let {
    id,
    label,
    value,
    icon,
    placeholder = '',
    min = undefined,
    step = '1',
    disabled = false,
    oninput,
  }: {
    id: string
    label: string
    value: string
    icon: Component
    placeholder?: string
    min?: string | undefined
    step?: string
    disabled?: boolean
    oninput?: (value: string) => void
  } = $props()

  const Icon = $derived(icon)
</script>

<div class="flex items-center gap-2 py-2">
  <Icon class="text-muted-foreground size-3.5 shrink-0" />
  <label for={id} class="text-muted-foreground min-w-0 shrink-0 text-xs">{label}</label>
  <Input
    {id}
    class="ml-auto h-7 w-20 text-xs"
    type="number"
    {min}
    {step}
    inputmode="numeric"
    {value}
    {placeholder}
    {disabled}
    oninput={(event) => oninput?.((event.currentTarget as HTMLInputElement).value)}
  />
</div>
