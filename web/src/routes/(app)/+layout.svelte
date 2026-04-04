<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte'
  import ProjectShell from '$lib/features/app-shell/components/project-shell.svelte'
  import type { Snippet } from 'svelte'
  import type { LayoutData } from './$types'

  let { children, data }: { children: Snippet; data: LayoutData } = $props()

  $effect(() => {
    authStore.hydrate(data.authSession)
  })
</script>

<ProjectShell {data}>
  {@render children()}
</ProjectShell>
