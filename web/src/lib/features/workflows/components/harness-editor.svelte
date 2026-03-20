<script lang="ts">
  import { cn } from '$lib/utils'
  import { FileCode, Copy, Check } from '@lucide/svelte'
  import type { HarnessContent } from '../types'

  let {
    content,
    filePath = '',
    version = 1,
    onchange,
    class: className = '',
  }: {
    content: HarnessContent
    filePath?: string
    version?: number
    onchange?: (raw: string) => void
    class?: string
  } = $props()

  let copied = $state(false)
  let lines = $derived(content.rawContent.split('\n'))

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement
    onchange?.(target.value)
  }

  async function copyContent() {
    await navigator.clipboard.writeText(content.rawContent)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }
</script>

<div class={cn('flex h-full flex-col overflow-hidden', className)}>
  <div class="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-2">
    <div class="flex items-center gap-2 text-sm">
      <FileCode class="size-4 text-muted-foreground" />
      <span class="font-mono text-xs text-muted-foreground">{filePath}</span>
      <span class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
        v{version}
      </span>
    </div>
    <button
      class="flex items-center gap-1 rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      onclick={copyContent}
    >
      {#if copied}
        <Check class="size-3" />
        Copied
      {:else}
        <Copy class="size-3" />
        Copy
      {/if}
    </button>
  </div>

  <div class="relative flex-1 overflow-hidden bg-[#0d1117]">
    <div class="flex h-full overflow-auto">
      <div
        class="sticky left-0 shrink-0 select-none border-r border-neutral-800 bg-[#0d1117] px-3 py-3 text-right font-mono text-xs leading-6 text-neutral-600"
        aria-hidden="true"
      >
        {#each lines as _, i}
          <div>{i + 1}</div>
        {/each}
      </div>

      <textarea
        class="min-h-full flex-1 resize-none bg-transparent p-3 font-mono text-xs leading-6 text-neutral-200 outline-none placeholder:text-neutral-600"
        spellcheck="false"
        value={content.rawContent}
        oninput={handleInput}
      ></textarea>
    </div>
  </div>
</div>
