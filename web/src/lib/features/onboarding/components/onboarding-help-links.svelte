<script lang="ts">
  import { Github, Terminal, FileText, BookOpen, ExternalLink } from '@lucide/svelte'

  let { onOpenProjectAI }: { onOpenProjectAI: (prompt: string) => void } = $props()
</script>

<div class="border-border bg-card rounded-xl border p-5">
  <h3 class="text-foreground mb-3 text-sm font-medium">帮助资源</h3>
  <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
    <a
      href="https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens"
      target="_blank"
      rel="noopener noreferrer"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
    >
      <Github class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">GitHub Token 帮助</span>
      <ExternalLink class="text-muted-foreground ml-auto size-3" />
    </a>
    <a
      href="https://docs.anthropic.com/en/docs/build-with-claude/computer-use"
      target="_blank"
      rel="noopener noreferrer"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
    >
      <Terminal class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">CLI 安装文档</span>
      <ExternalLink class="text-muted-foreground ml-auto size-3" />
    </a>
    <button
      type="button"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
      onclick={() => onOpenProjectAI('帮我看看项目的 Harness 示例')}
    >
      <FileText class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">示例 Harness</span>
    </button>
    <button
      type="button"
      class="border-border hover:bg-muted/50 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors"
      onclick={() => onOpenProjectAI('教我如何使用 CLI 创建和管理工单')}
    >
      <BookOpen class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-foreground">CLI 使用示例</span>
    </button>
  </div>
</div>
