<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    stage,
    color = '#94a3b8',
    class: className = '',
  }: {
    stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
    color?: string
    class?: string
  } = $props()
</script>

<svg
  viewBox="0 0 16 16"
  fill="none"
  xmlns="http://www.w3.org/2000/svg"
  class={cn('size-4 shrink-0', className)}
  aria-hidden="true"
>
  {#if stage === 'backlog'}
    <!-- Dashed circle, hollow -->
    <circle cx="8" cy="8" r="6" stroke={color} stroke-width="1.5" stroke-dasharray="3 2" />
  {:else if stage === 'unstarted'}
    <!-- Thin solid circle, hollow -->
    <circle cx="8" cy="8" r="6" stroke={color} stroke-width="1.5" />
  {:else if stage === 'started'}
    <!-- Thick circle with pie wedge fill (50%) -->
    <circle cx="8" cy="8" r="6" stroke={color} stroke-width="2" />
    <path d="M8 2 A6 6 0 0 1 8 14 Z" fill={color} opacity="0.5" />
  {:else if stage === 'completed'}
    <!-- Filled circle with checkmark -->
    <circle cx="8" cy="8" r="7" fill={color} />
    <path
      d="M5 8.2 L7 10.2 L11 6"
      stroke="white"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      fill="none"
    />
  {:else if stage === 'canceled'}
    <!-- Circle with X -->
    <circle cx="8" cy="8" r="6" stroke={color} stroke-width="1.5" />
    <path
      d="M5.75 5.75 L10.25 10.25 M10.25 5.75 L5.75 10.25"
      stroke={color}
      stroke-width="1.5"
      stroke-linecap="round"
    />
  {/if}
</svg>
