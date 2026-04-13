<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { Snippet } from 'svelte'

  let { children }: { children?: Snippet } = $props()

  $effect(() => {
    if (typeof document === 'undefined') {
      return
    }

    document.documentElement.classList.toggle('dark', appStore.theme === 'dark')
    document.documentElement.lang = i18nStore.locale

    return () => {
      document.documentElement.classList.remove('dark')
    }
  })

  $effect(() => {
    i18nStore.init()
  })
</script>

<div class:dark={appStore.theme === 'dark'}>
  {#if children}
    {@render children()}
  {/if}
</div>
