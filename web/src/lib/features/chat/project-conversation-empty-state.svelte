<script lang="ts">
  import type { Component } from 'svelte'
  import {
    ArrowUpRight,
    FolderGit2,
    GitBranch,
    LayoutList,
    PencilRuler,
    Puzzle,
    Sparkles,
    TimerReset,
    TrendingUp,
  } from '@lucide/svelte'
  import type { TranslationKey } from '$lib/i18n'
  import { chatT } from './i18n'

  let {
    onPickPrompt,
  }: {
    onPickPrompt?: (text: string) => void
  } = $props()

  type Suggestion = {
    key: TranslationKey
    icon: Component
    tone: string
  }

  const suggestions: Suggestion[] = [
    {
      key: 'chat.emptyState.suggestions.workflows',
      icon: GitBranch,
      tone: 'text-sky-500',
    },
    {
      key: 'chat.emptyState.suggestions.ticketTrends',
      icon: TrendingUp,
      tone: 'text-emerald-500',
    },
    {
      key: 'chat.emptyState.suggestions.runningAgents',
      icon: LayoutList,
      tone: 'text-amber-500',
    },
    {
      key: 'chat.emptyState.suggestions.harnessDraft',
      icon: PencilRuler,
      tone: 'text-violet-500',
    },
    {
      key: 'chat.emptyState.suggestions.blockedTickets',
      icon: TimerReset,
      tone: 'text-rose-500',
    },
    {
      key: 'chat.emptyState.suggestions.reusableSkills',
      icon: Puzzle,
      tone: 'text-indigo-500',
    },
  ]
</script>

<div class="animate-fade-in-up flex flex-col gap-6 px-4 py-8">
  <!-- Hero -->
  <div class="relative flex flex-col items-center text-center">
    <div
      class="from-primary/25 pointer-events-none absolute inset-x-0 top-0 h-28 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-from),_transparent_70%)] blur-2xl"
      aria-hidden="true"
    ></div>

    <div class="relative flex size-10 items-center justify-center">
      <Sparkles
        class="text-primary size-8 drop-shadow-[0_0_10px_color-mix(in_oklab,var(--primary)_55%,transparent)]"
      />
    </div>

    <div
      class="from-foreground to-foreground/70 mt-3 bg-gradient-to-b bg-clip-text text-xl font-semibold tracking-tight text-transparent"
    >
      {chatT('chat.emptyState.title')}
    </div>
    <p class="text-muted-foreground mt-1.5 max-w-xs text-[13px] leading-relaxed">
      {chatT('chat.emptyState.description')}
    </p>
  </div>

  <!-- Suggestions -->
  <div class="space-y-2">
    <div class="flex items-center gap-2">
      <div class="via-border h-px flex-1 bg-gradient-to-r from-transparent to-transparent"></div>
      <div
        class="text-muted-foreground flex items-center gap-1 text-[10px] font-semibold tracking-[0.14em] uppercase"
      >
        <Sparkles class="size-2.5" />
        {chatT('chat.emptyState.suggestions.label')}
      </div>
      <div class="via-border h-px flex-1 bg-gradient-to-r from-transparent to-transparent"></div>
    </div>

    <div class="grid grid-cols-1 gap-1.5" data-tour="project-ai-suggestions">
      {#each suggestions as item}
        {@const prompt = chatT(item.key)}
        {@const Icon = item.icon}
        <button
          type="button"
          class="group border-border/60 bg-card/40 hover:border-primary/40 hover:bg-muted/60 relative flex items-center gap-3 overflow-hidden rounded-xl border px-3 py-2.5 text-left transition-all hover:shadow-sm"
          onclick={() => onPickPrompt?.(prompt)}
        >
          <div
            class="bg-muted/70 group-hover:bg-background flex size-8 shrink-0 items-center justify-center rounded-lg transition-colors"
          >
            <Icon class="size-4 {item.tone}" />
          </div>
          <span class="text-foreground min-w-0 flex-1 text-[13px] leading-snug">
            {prompt}
          </span>
          <ArrowUpRight
            class="text-muted-foreground/40 group-hover:text-foreground size-3.5 shrink-0 translate-y-0 transition-all group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
          />
        </button>
      {/each}
    </div>
  </div>

  <!-- Isolation footer -->
  <div
    class="border-border/50 bg-muted/20 text-muted-foreground flex items-start gap-2.5 rounded-lg border border-dashed px-3 py-2.5 text-[11px] leading-relaxed"
    data-tour="project-ai-isolation"
  >
    <FolderGit2 class="text-muted-foreground/80 mt-0.5 size-3.5 shrink-0" />
    <div>
      <span class="text-foreground font-medium">
        {chatT('chat.emptyState.isolation.title')}
      </span>
      <span class="ml-1">{chatT('chat.emptyState.isolation.description')}</span>
    </div>
  </div>
</div>
