<script lang="ts">
  import ThemeRoot from '$lib/components/layout/theme-root.svelte'
  import { ToastContainer } from '$ui/toast'
  import './layout.css'

  let { children } = $props()
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
  <link
    href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Noto+Sans+SC:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap"
    rel="stylesheet"
  />
</svelte:head>

<ThemeRoot>
  {@render children()}
  <ToastContainer />
</ThemeRoot>
