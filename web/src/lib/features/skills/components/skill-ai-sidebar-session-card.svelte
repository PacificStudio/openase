<script lang="ts">
  import {
    ProjectConversationTranscript,
    type ProjectConversationTranscriptEntry,
  } from '$lib/features/chat'
  import { Badge } from '$ui/badge'
  import { LoaderCircle, ShieldCheck, ShieldX, Wrench } from '@lucide/svelte'
  import type { SkillRefinementAnchorState } from './skill-refinement-transcript'

  let {
    phase,
    phaseMessage,
    attempt,
    workspacePath,
    anchorState,
    transcriptEntries,
    pending,
  }: {
    phase: '' | 'editing' | 'testing' | 'retrying' | 'verified' | 'blocked' | 'unverified'
    phaseMessage: string
    attempt: number
    workspacePath: string
    anchorState: SkillRefinementAnchorState
    transcriptEntries: ProjectConversationTranscriptEntry[]
    pending: boolean
  } = $props()

  const providerAnchorLabel = $derived.by(() => {
    const kind = anchorState.anchorKind?.trim()
    if (kind === 'session') return 'Provider Session'
    if (kind === 'thread') return 'Provider Thread'
    return 'Provider Anchor'
  })

  function phaseBadgeClass(value: typeof phase) {
    switch (value) {
      case 'verified':
        return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200'
      case 'blocked':
        return 'border-rose-500/40 bg-rose-500/10 text-rose-200'
      case 'unverified':
        return 'border-amber-500/40 bg-amber-500/10 text-amber-200'
      case 'testing':
        return 'border-sky-500/40 bg-sky-500/10 text-sky-200'
      case 'retrying':
        return 'border-orange-500/40 bg-orange-500/10 text-orange-200'
      default:
        return 'border-primary/30 bg-primary/10 text-foreground'
    }
  }

  function phaseLabel(value: typeof phase) {
    switch (value) {
      case 'editing':
        return 'Editing'
      case 'testing':
        return 'Testing'
      case 'retrying':
        return 'Retrying'
      case 'verified':
        return 'Verified'
      case 'blocked':
        return 'Blocked'
      case 'unverified':
        return 'Unverified'
      default:
        return 'Idle'
    }
  }
</script>

<div class="space-y-3">
  <div class="rounded-lg border border-white/8 bg-white/4 p-3">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <Badge variant="outline" class={phaseBadgeClass(phase)}>
            {phaseLabel(phase)}
          </Badge>
          {#if attempt > 0}
            <span class="text-muted-foreground text-[11px]">Attempt {attempt}</span>
          {/if}
        </div>
        <p class="text-muted-foreground text-xs leading-5">
          {phaseMessage ||
            'Ask Codex to edit the draft bundle and verify it in an isolated workspace.'}
        </p>
      </div>
      {#if phase === 'verified'}
        <ShieldCheck class="mt-0.5 size-4 shrink-0 text-emerald-300" />
      {:else if phase === 'blocked' || phase === 'unverified'}
        <ShieldX class="mt-0.5 size-4 shrink-0 text-rose-300" />
      {:else}
        <Wrench class="mt-0.5 size-4 shrink-0 text-sky-300" />
      {/if}
    </div>

    {#if workspacePath}
      <div class="mt-3 rounded-md border border-white/6 bg-black/20 px-2.5 py-2">
        <p class="text-muted-foreground text-[10px] tracking-[0.18em] uppercase">Workspace</p>
        <p class="mt-1 font-mono text-[11px] leading-5 break-all">{workspacePath}</p>
      </div>
    {/if}

    {#if anchorState.anchorId}
      <div class="mt-3 rounded-md border border-white/6 bg-black/20 px-2.5 py-2">
        <p class="text-muted-foreground text-[10px] tracking-[0.18em] uppercase">
          {providerAnchorLabel}
        </p>
        <p class="mt-1 font-mono text-[11px] leading-5 break-all">{anchorState.anchorId}</p>
        {#if anchorState.turnId}
          <p class="text-muted-foreground mt-1 text-[11px] leading-5 break-all">
            turn: <span class="font-mono">{anchorState.turnId}</span>
          </p>
        {/if}
      </div>
    {/if}
  </div>

  {#if pending && phase === 'editing'}
    <div
      class="flex items-center gap-2 rounded-md bg-sky-500/10 px-2.5 py-1.5 text-[11px] text-sky-300"
    >
      <LoaderCircle class="size-3 shrink-0 animate-spin" />
      Suggesting diff...
    </div>
  {/if}

  {#if transcriptEntries.length > 0}
    <div class="rounded-lg border border-white/8 bg-white/3 p-2">
      <div class="mb-2 px-1">
        <p class="text-muted-foreground text-[10px] tracking-[0.18em] uppercase">Transcript</p>
      </div>
      <ProjectConversationTranscript entries={transcriptEntries} {pending} />
    </div>
  {/if}
</div>
