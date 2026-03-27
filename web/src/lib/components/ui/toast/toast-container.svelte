<script lang="ts">
  import { toastStore, type Toast } from '$lib/stores/toast.svelte'
  import X from '@lucide/svelte/icons/x'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import CircleX from '@lucide/svelte/icons/circle-x'
  import Info from '@lucide/svelte/icons/info'
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert'
  import { cn } from '$lib/utils'

  let hovered = $state(false)
  let timers = new Map<string, ReturnType<typeof setTimeout>>()

  const variantConfig = {
    success: {
      icon: CircleCheck,
      containerClass: 'border-success/30 bg-success/10',
      iconClass: 'text-success',
      textClass: 'text-foreground',
    },
    error: {
      icon: CircleX,
      containerClass: 'border-destructive/30 bg-destructive/10',
      iconClass: 'text-destructive',
      textClass: 'text-foreground',
    },
    warning: {
      icon: TriangleAlert,
      containerClass: 'border-warning/30 bg-warning/10',
      iconClass: 'text-warning',
      textClass: 'text-foreground',
    },
    info: {
      icon: Info,
      containerClass: 'border-info/30 bg-info/10',
      iconClass: 'text-info',
      textClass: 'text-foreground',
    },
  }

  function scheduleRemoval(toast: Toast) {
    if (timers.has(toast.id)) return
    const timer = setTimeout(() => {
      timers.delete(toast.id)
      toastStore.dismiss(toast.id)
    }, toast.duration)
    timers.set(toast.id, timer)
  }

  function pauseAll() {
    for (const [id, timer] of timers) {
      clearTimeout(timer)
      timers.delete(id)
    }
  }

  function resumeAll() {
    for (const toast of toastStore.toasts) {
      const elapsed = Date.now() - toast.createdAt
      const remaining = Math.max(toast.duration - elapsed, 500)
      if (timers.has(toast.id)) continue
      const timer = setTimeout(() => {
        timers.delete(toast.id)
        toastStore.dismiss(toast.id)
      }, remaining)
      timers.set(toast.id, timer)
    }
  }

  function handleMouseEnter() {
    hovered = true
    pauseAll()
  }

  function handleMouseLeave() {
    hovered = false
    resumeAll()
  }

  $effect(() => {
    const currentToasts = toastStore.toasts
    if (hovered) return

    for (const toast of currentToasts) {
      scheduleRemoval(toast)
    }
  })
</script>

{#if toastStore.toasts.length > 0}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed right-4 bottom-4 z-[9999] flex flex-col-reverse items-end"
    onmouseenter={handleMouseEnter}
    onmouseleave={handleMouseLeave}
  >
    {#each toastStore.toasts as toast, i (toast.id)}
      {@const config = variantConfig[toast.variant]}
      {@const total = toastStore.toasts.length}
      {@const reverseIndex = total - 1 - i}
      <div
        class={cn(
          'pointer-events-auto w-80 rounded-lg border px-4 py-3 shadow-lg backdrop-blur-sm transition-all duration-300',
          config.containerClass,
          !hovered && reverseIndex > 0 && 'absolute',
        )}
        style={!hovered && reverseIndex > 0
          ? `bottom: ${reverseIndex * 6}px; right: ${reverseIndex * 2}px; transform: scale(${1 - reverseIndex * 0.03}); opacity: ${Math.max(1 - reverseIndex * 0.2, 0.3)}; z-index: ${total - reverseIndex};`
          : `position: relative; z-index: ${total - reverseIndex}; margin-bottom: ${i < total - 1 ? '8px' : '0'};`}
      >
        <div class="flex items-start gap-3">
          <config.icon class={cn('mt-0.5 size-4 shrink-0', config.iconClass)} />
          <p class={cn('flex-1 text-sm leading-snug', config.textClass)}>{toast.message}</p>
          <button
            class="text-muted-foreground hover:text-foreground -mt-0.5 -mr-1 shrink-0 rounded p-0.5 transition-colors"
            onclick={() => toastStore.dismiss(toast.id)}
            aria-label="Dismiss"
          >
            <X class="size-3.5" />
          </button>
        </div>
      </div>
    {/each}
  </div>
{/if}
