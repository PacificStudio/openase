<script lang="ts">
  import type { Snippet } from 'svelte'
  import { PanelRightOpen } from '@lucide/svelte'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'

  let {
    children,
    title = 'Workspace drawer',
    description = 'Keep project setup, workflow tuning, and harness editing off the route layer.',
  }: {
    children?: Snippet
    title?: string
    description?: string
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 sticky top-5 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <PanelRightOpen class="size-4" />
      <span>{title}</span>
    </CardTitle>
    <CardDescription>{description}</CardDescription>
  </CardHeader>
  <CardContent class="space-y-6">
    {@render children?.()}
  </CardContent>
</Card>
