<script lang="ts">
  import type { Snippet } from 'svelte'
  import { cn } from '$lib/utils'
  import type { ProjectSection } from '$lib/stores/app-context'
  import PageHeader from './page-header.svelte'

  let {
    title,
    description = '',
    actions,
    helpSection,
    variant = 'flow',
    class: className = '',
    contentClass = '',
    children,
  }: {
    title: string
    description?: string
    actions?: Snippet
    helpSection?: ProjectSection
    variant?: 'flow' | 'workspace'
    class?: string
    contentClass?: string
    children: Snippet
  } = $props()
</script>

<div class={cn('flex h-full min-h-0 flex-col', className)}>
  <PageHeader {title} {description} {actions} {helpSection} />
  <div
    data-testid="page-scaffold-content"
    class={cn(
      'min-h-0 flex-1 px-4 py-4 sm:px-6',
      variant === 'workspace' && 'flex flex-col overflow-hidden',
      variant === 'flow' && 'overflow-y-auto',
      contentClass,
    )}
  >
    {@render children()}
  </div>
</div>
