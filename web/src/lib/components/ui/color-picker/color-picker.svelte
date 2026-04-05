<script lang="ts">
  import { cn } from '$lib/utils'
  import * as Popover from '$ui/popover'
  import { ChevronDown, ChevronUp, Pipette, Check } from '@lucide/svelte'
  import { quickPicks, expandedPalette, isPresetColor } from './palette'

  let {
    value = $bindable('#3b82f6'),
    disabled = false,
    class: className,
  }: {
    value?: string
    disabled?: boolean
    class?: string
  } = $props()

  let open = $state(false)
  let showExpanded = $state(false)
  let showCustom = $state(false)
  let hexInput = $state('')

  const normalizedValue = $derived(value.toLowerCase())

  function selectColor(hex: string) {
    value = hex.toLowerCase()
    open = false
    showExpanded = false
    showCustom = false
  }

  function handleHexInput() {
    const cleaned = hexInput.trim().toLowerCase()
    const hex = cleaned.startsWith('#') ? cleaned : `#${cleaned}`
    if (/^#[0-9a-f]{6}$/.test(hex)) {
      value = hex
      open = false
      showExpanded = false
      showCustom = false
    }
  }

  function handleNativeColorChange(e: Event) {
    const target = e.target as HTMLInputElement
    value = target.value.toLowerCase()
  }

  function handleKeydown(hex: string, e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      selectColor(hex)
    }
  }

  function handleOpen(isOpen: boolean) {
    open = isOpen
    if (isOpen) {
      hexInput = normalizedValue
      showExpanded = false
      showCustom = false
    }
  }
</script>

<Popover.Root onOpenChange={handleOpen} bind:open>
  <Popover.Trigger>
    {#snippet child({ props })}
      <button
        {...props}
        type="button"
        {disabled}
        class={cn(
          'focus-visible:ring-ring inline-flex size-8 shrink-0 items-center justify-center rounded border border-transparent ring-1 ring-black/10 transition-shadow hover:ring-black/20 focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50',
          className,
        )}
        style="background-color: {normalizedValue}"
        aria-label="Pick a color: {normalizedValue}"
      >
        <span class="sr-only">{normalizedValue}</span>
      </button>
    {/snippet}
  </Popover.Trigger>

  <Popover.Content class="w-auto min-w-0 p-3" align="start" sideOffset={6}>
    <!-- Quick picks (always visible) -->
    <div class="flex items-center gap-1.5" role="radiogroup" aria-label="Quick color picks">
      {#each quickPicks as hex (hex)}
        <button
          type="button"
          role="radio"
          aria-checked={normalizedValue === hex}
          aria-label={hex}
          class={cn(
            'focus-visible:ring-ring relative size-7 shrink-0 rounded transition-all focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:outline-none',
            normalizedValue === hex
              ? 'ring-foreground ring-offset-background scale-110 ring-2 ring-offset-2'
              : 'ring-1 ring-black/10 hover:scale-105 hover:ring-black/25',
          )}
          style="background-color: {hex}"
          onclick={() => selectColor(hex)}
          onkeydown={(e) => handleKeydown(hex, e)}
        >
          {#if normalizedValue === hex}
            <Check
              class="absolute inset-0 m-auto size-3.5 drop-shadow-[0_1px_1px_rgba(0,0,0,0.5)]"
              style="color: white"
              strokeWidth={3}
            />
          {/if}
        </button>
      {/each}
    </div>

    <!-- Expand / collapse toggle -->
    <button
      type="button"
      class="text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-ring mt-2.5 flex w-full items-center justify-center gap-1 rounded py-1 text-xs transition-colors focus-visible:ring-2 focus-visible:outline-none"
      onclick={() => {
        showExpanded = !showExpanded
        if (!showExpanded) showCustom = false
      }}
      aria-expanded={showExpanded}
    >
      {#if showExpanded}
        <ChevronUp class="size-3" />
        Fewer colors
      {:else}
        <ChevronDown class="size-3" />
        More colors
      {/if}
    </button>

    <!-- Expanded preset grid (5 × 8) -->
    {#if showExpanded}
      <div class="mt-2 space-y-1" role="radiogroup" aria-label="Extended color palette">
        {#each expandedPalette as row, rowIdx (rowIdx)}
          <div class="flex items-center gap-1">
            {#each row as hex (hex)}
              <button
                type="button"
                role="radio"
                aria-checked={normalizedValue === hex}
                aria-label={hex}
                class={cn(
                  'focus-visible:ring-ring relative size-6 shrink-0 rounded-sm transition-all focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:outline-none',
                  normalizedValue === hex
                    ? 'ring-foreground ring-offset-background z-10 scale-110 ring-2 ring-offset-1'
                    : 'ring-1 ring-black/8 hover:z-10 hover:scale-110 hover:ring-black/20',
                )}
                style="background-color: {hex}"
                onclick={() => selectColor(hex)}
                onkeydown={(e) => handleKeydown(hex, e)}
              >
                {#if normalizedValue === hex}
                  <Check
                    class="absolute inset-0 m-auto size-3 drop-shadow-[0_1px_1px_rgba(0,0,0,0.5)]"
                    style="color: white"
                    strokeWidth={3}
                  />
                {/if}
              </button>
            {/each}
          </div>
        {/each}
      </div>

      <!-- Custom color section -->
      <div class="border-border mt-2.5 border-t pt-2.5">
        {#if !showCustom}
          <button
            type="button"
            class="text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-ring flex w-full items-center justify-center gap-1.5 rounded py-1 text-xs transition-colors focus-visible:ring-2 focus-visible:outline-none"
            onclick={() => {
              showCustom = true
              hexInput = normalizedValue
            }}
          >
            <Pipette class="size-3" />
            Custom color
          </button>
        {:else}
          <div class="flex items-center gap-2">
            <div
              class="size-7 shrink-0 rounded ring-1 ring-black/10"
              style="background-color: {normalizedValue}"
            ></div>
            <div class="relative flex-1">
              <input
                type="text"
                bind:value={hexInput}
                maxlength={7}
                placeholder="#000000"
                class="border-input bg-background text-foreground placeholder:text-muted-foreground focus-visible:ring-ring h-7 w-full rounded border px-2 font-mono text-xs focus-visible:ring-2 focus-visible:outline-none"
                onkeydown={(e) => {
                  if (e.key === 'Enter') handleHexInput()
                }}
              />
            </div>
            <label
              class="border-input bg-background hover:bg-muted inline-flex size-7 shrink-0 cursor-pointer items-center justify-center rounded border transition-colors"
            >
              <Pipette class="text-muted-foreground size-3.5" />
              <input
                type="color"
                value={normalizedValue}
                class="sr-only"
                onchange={handleNativeColorChange}
              />
            </label>
            <button
              type="button"
              class="bg-primary text-primary-foreground hover:bg-primary/90 focus-visible:ring-ring h-7 shrink-0 rounded px-2.5 text-xs font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none"
              onclick={handleHexInput}
            >
              Apply
            </button>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Show current value if it's not a preset -->
    {#if !isPresetColor(normalizedValue) && !showCustom}
      <div class="border-border mt-2 flex items-center gap-1.5 border-t pt-2">
        <div
          class="ring-foreground ring-offset-background size-5 shrink-0 rounded-sm ring-2 ring-offset-1"
          style="background-color: {normalizedValue}"
        ></div>
        <span class="text-muted-foreground font-mono text-xs">{normalizedValue}</span>
      </div>
    {/if}
  </Popover.Content>
</Popover.Root>
