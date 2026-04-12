<script lang="ts">
  import { Github, Terminal, FileText, BookOpen, ExternalLink } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let { onOpenProjectAI }: { onOpenProjectAI: (prompt: string) => void } = $props()
</script>

<div class="border-border bg-card rounded-xl border p-5">
  <h3 class="text-foreground mb-3 text-sm font-medium">
    {i18nStore.t('onboarding.helpLinks.heading')}
  </h3>
  <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
    <a
      href="https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens"
      target="_blank"
      rel="noopener noreferrer"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
    >
      <Github class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">
        {i18nStore.t('onboarding.helpLinks.githubToken')}
      </span>
      <ExternalLink class="text-muted-foreground ml-auto size-3" />
    </a>
    <a
      href="https://docs.anthropic.com/en/docs/build-with-claude/computer-use"
      target="_blank"
      rel="noopener noreferrer"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
    >
      <Terminal class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">
        {i18nStore.t('onboarding.helpLinks.cliInstall')}
      </span>
      <ExternalLink class="text-muted-foreground ml-auto size-3" />
    </a>
    <button
      type="button"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
      onclick={() => onOpenProjectAI('Show me sample harnesses for this project.')}
    >
      <FileText class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">
        {i18nStore.t('onboarding.helpLinks.harnessExamples')}
      </span>
    </button>
    <button
      type="button"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
      onclick={() => onOpenProjectAI('Show me how to create and manage tickets with the CLI.')}
    >
      <BookOpen class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">
        {i18nStore.t('onboarding.helpLinks.cliExamples')}
      </span>
    </button>
  </div>
</div>
