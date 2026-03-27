<script lang="ts">
  import { Checkbox as CheckboxPrimitive } from 'bits-ui'
  import CheckIcon from '@lucide/svelte/icons/check'
  import MinusIcon from '@lucide/svelte/icons/minus'
  import { cn, type WithoutChildrenOrChild } from '$lib/utils.js'

  let {
    ref = $bindable(null),
    checked = $bindable(false),
    indeterminate = $bindable(false),
    class: className,
    ...restProps
  }: WithoutChildrenOrChild<CheckboxPrimitive.RootProps> = $props()
</script>

<CheckboxPrimitive.Root
  bind:ref
  bind:checked
  bind:indeterminate
  data-slot="checkbox"
  class={cn(
    'border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 data-[state=checked]:bg-primary data-[state=checked]:border-primary data-[state=checked]:text-primary-foreground data-[state=indeterminate]:bg-primary data-[state=indeterminate]:border-primary data-[state=indeterminate]:text-primary-foreground bg-background inline-flex size-4 shrink-0 items-center justify-center rounded-[4px] border shadow-xs transition-[color,box-shadow] outline-none focus-visible:ring-3 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50',
    className,
  )}
  {...restProps}
>
  {#snippet children({ checked, indeterminate })}
    <span class="pointer-events-none flex items-center justify-center text-current">
      {#if indeterminate}
        <MinusIcon class="size-3.5" />
      {:else if checked}
        <CheckIcon class="size-3.5" />
      {/if}
    </span>
  {/snippet}
</CheckboxPrimitive.Root>
