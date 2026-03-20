<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import type { Snippet } from 'svelte'

  let { children }: { children?: Snippet } = $props()

  $effect(() => {
    if (typeof document === 'undefined') {
      return
    }

    document.documentElement.classList.toggle('dark', appStore.theme === 'dark')

    return () => {
      document.documentElement.classList.remove('dark')
    }
  })
</script>

<div class:dark={appStore.theme === 'dark'}>
  {#if children}
    {@render children()}
  {/if}
</div>
