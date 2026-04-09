<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronDown, ChevronUp } from '@lucide/svelte'
  import ProjectUpdateMarkdownContent from './project-update-markdown-content.svelte'

  let {
    bodyMarkdown,
    title,
    isDeleted = false,
  }: {
    bodyMarkdown?: string
    title?: string
    isDeleted?: boolean
  } = $props()

  let expanded = $state(false)
  let contentRef = $state<HTMLDivElement | null>(null)
  let overflows = $state(false)

  $effect(() => {
    if (contentRef) {
      const lineHeight = 20
      overflows = contentRef.scrollHeight > lineHeight * 3 + 4
    }
  })
</script>

{#if bodyMarkdown}
  <div
    bind:this={contentRef}
    class={cn('mt-1 overflow-hidden', !expanded && overflows && 'max-h-[64px]')}
  >
    <ProjectUpdateMarkdownContent
      source={bodyMarkdown}
      class={cn('text-sm leading-5', isDeleted && 'line-through opacity-60')}
    />
  </div>
  {#if overflows}
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground mt-0.5 flex items-center gap-0.5 text-xs transition-colors"
      onclick={() => (expanded = !expanded)}
    >
      {#if expanded}<ChevronUp class="size-3" />Show less{:else}<ChevronDown class="size-3" />Show
        more{/if}
    </button>
  {/if}
{:else if title}
  <p class={cn('mt-1 text-sm', isDeleted && 'line-through opacity-60')}>{title}</p>
{/if}
