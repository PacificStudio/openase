<script lang="ts">
  import ProjectShell from '$lib/features/app-shell/components/project-shell.svelte'
  import type { Snippet } from 'svelte'
  import type { LayoutData } from './$types'

  let { children, data }: { children: Snippet; data: LayoutData } = $props()
</script>

<ProjectShell {data}>
  {@render children()}
</ProjectShell>
