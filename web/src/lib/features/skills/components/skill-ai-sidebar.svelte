<script lang="ts">
  import type { AgentProvider, SkillFile } from '$lib/api/contracts'
  import { EphemeralChatProviderSelect } from '$lib/features/chat'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Plus, ShieldCheck } from '@lucide/svelte'
  import { createSkillAISidebarController } from './skill-ai-sidebar-controller.svelte'
  import SkillAISidebarResultPanel from './skill-ai-sidebar-result-panel.svelte'
  import SkillAISidebarSessionCard from './skill-ai-sidebar-session-card.svelte'

  let {
    projectId,
    providers = [],
    skillId,
    files = [],
    onApplySuggestion,
  }: {
    projectId?: string
    providers?: AgentProvider[]
    skillId?: string
    files?: SkillFile[]
    onApplySuggestion?: (files: SkillFile[], focusPath?: string) => void
  } = $props()

  const controller = createSkillAISidebarController({
    getProjectId: () => projectId,
    getProviders: () => providers,
    getSkillId: () => skillId,
    getFiles: () => files,
    onApplySuggestion: (nextFiles, focusPath) => onApplySuggestion?.(nextFiles, focusPath),
  })
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-3 py-1">
    <div class="flex min-w-0 items-center gap-1.5">
      <span class="text-muted-foreground text-[11px] font-medium">Fix & verify</span>
      <EphemeralChatProviderSelect
        providers={controller.refinementProviders}
        capability="skill_ai"
        providerId={controller.providerId}
        onProviderChange={(nextProviderId) => void controller.handleProviderChange(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-6 p-0"
      aria-label="Reset refinement run"
      onclick={() => void controller.closeActiveSession({ clearResult: true })}
      disabled={!controller.sessionId && !controller.result && !controller.pending}
    >
      <Plus class="size-3" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-3 py-2">
    <div class="space-y-3">
      <SkillAISidebarSessionCard
        phase={controller.phase}
        phaseMessage={controller.phaseMessage}
        attempt={controller.attempt}
        workspacePath={controller.workspacePath}
        anchorState={controller.anchorState}
        transcriptEntries={controller.transcriptEntries}
        pending={controller.pending}
      />

      <SkillAISidebarResultPanel
        result={controller.result}
        suggestion={controller.suggestion}
        preview={controller.preview}
        dismissed={controller.dismissed}
        selectedSuggestionPath={controller.selectedSuggestionPath}
        suggestionAlreadyApplied={controller.suggestionAlreadyApplied}
        onSelectPath={controller.selectSuggestionPath}
        onApply={controller.handleApply}
        onDismiss={controller.handleDismiss}
      />
    </div>
  </ScrollArea>

  <div class="border-border border-t px-3 py-2">
    <div class="space-y-2">
      <Textarea
        value={controller.prompt}
        rows={4}
        class="min-h-[88px] resize-none text-xs"
        placeholder="Describe what Codex should improve and verify for this draft bundle…"
        disabled={!projectId || !skillId || !controller.providerId || controller.pending}
        oninput={(event) =>
          controller.setPrompt((event.currentTarget as HTMLTextAreaElement).value)}
        onkeydown={controller.handlePromptKeydown}
      />
      <Button
        class="h-8 w-full gap-2 text-xs"
        onclick={() => void controller.handleSend()}
        disabled={controller.sendDisabled}
      >
        <ShieldCheck class="size-3.5" />
        {controller.pending ? 'Running verification…' : 'Fix and verify'}
      </Button>
    </div>
  </div>
</div>
