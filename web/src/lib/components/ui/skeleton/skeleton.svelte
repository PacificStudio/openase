<script lang="ts">
  import { cn } from '$lib/utils'

  let { class: className = '' }: { class?: string } = $props()
</script>

<div class={cn('bg-muted animate-shimmer animate-pulse rounded-md', className)}></div>
