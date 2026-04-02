<script lang="ts">
  /* eslint-disable max-lines */
  import { untrack } from 'svelte'
  import {
    closeSkillRefinementSession,
    streamSkillRefinement,
    type SkillRefinementResultPayload,
  } from '$lib/api/skill-refinement'
  import type { AgentProvider, SkillFile } from '$lib/api/contracts'
  import { EphemeralChatProviderSelect } from '$lib/features/chat'
  import {
    listProviderCapabilityProviders,
    pickDefaultProviderCapability,
    shouldKeepProviderCapability,
  } from '$lib/features/chat'
  import { buildDiffPreview } from '$lib/features/skills/assistant'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { LoaderCircle, Plus, ShieldCheck, ShieldX, Sparkles, Wrench } from '@lucide/svelte'
  import SkillSuggestionCard from './skill-suggestion-card.svelte'
  import { encodeUTF8Base64 } from './skill-bundle-editor'

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

  let prompt = $state('')
  let refinementProviders = $state<AgentProvider[]>([])
  let providerId = $state('')
  let pending = $state(false)
  let sessionId = $state('')
  let workspacePath = $state('')
  let phase = $state<
    '' | 'editing' | 'testing' | 'retrying' | 'verified' | 'blocked' | 'unverified'
  >('')
  let phaseMessage = $state('')
  let attempt = $state(0)
  let result = $state<SkillRefinementResultPayload | null>(null)
  let selectedSuggestionPath = $state('')
  let appliedBundleHash = $state('')
  let dismissed = $state(false)
  let previousContextKey = ''
  let abortController: AbortController | null = null

  const textPreviewFiles = $derived(
    result?.candidateFiles.filter(
      (file) => file.encoding === 'utf8' && typeof file.content === 'string',
    ) ?? [],
  )
  const suggestion = $derived(
    result?.status === 'verified' && textPreviewFiles.length > 0
      ? {
          summary: result.transcriptSummary || 'Codex verified this draft bundle.',
          files: textPreviewFiles.map((file) => ({
            path: file.path,
            content: file.content ?? '',
          })),
        }
      : null,
  )
  const previewTarget = $derived(
    suggestion?.files.find((file) => file.path === selectedSuggestionPath) ??
      suggestion?.files[0] ??
      null,
  )
  const preview = $derived(
    previewTarget
      ? buildDiffPreview(
          files.find((file) => file.path === previewTarget.path)?.content ?? '',
          previewTarget.content,
        )
      : null,
  )
  const previewList = $derived(
    suggestion?.files.map((file) => ({
      path: file.path,
      preview: buildDiffPreview(
        files.find((current) => current.path === file.path)?.content ?? '',
        file.content,
      ),
    })) ?? [],
  )
  const suggestionAlreadyApplied = $derived(
    (result?.candidateBundleHash && appliedBundleHash === result.candidateBundleHash) ||
      (previewList.length > 0 && previewList.every((item) => !item.preview.hasChanges)),
  )
  const sendDisabled = $derived(!projectId || !skillId || !providerId || !prompt.trim() || pending)

  $effect(() => {
    const nextProviders = providers
    const nextDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''
    untrack(() => {
      refinementProviders = listProviderCapabilityProviders(nextProviders, 'skill_ai')
      if (shouldKeepProviderCapability(refinementProviders, providerId, 'skill_ai')) {
        return
      }
      const nextProviderId = pickDefaultProviderCapability(
        refinementProviders,
        nextDefaultProviderId,
        'skill_ai',
      )
      if (providerId && providerId !== nextProviderId) {
        void closeActiveSession({ clearResult: true, suppressError: true })
      }
      providerId = nextProviderId
    })
  })

  $effect(() => {
    const contextKey = projectId && skillId ? `${projectId}:${skillId}` : ''
    if (contextKey === previousContextKey) {
      return
    }
    previousContextKey = contextKey
    prompt = ''
    appliedBundleHash = ''
    dismissed = false
    selectedSuggestionPath = ''
    void closeActiveSession({ clearResult: true, suppressError: true })
  })

  $effect(() => {
    if (!suggestion || suggestion.files.length === 0) {
      selectedSuggestionPath = ''
      return
    }
    const stillExists = suggestion.files.some((file) => file.path === selectedSuggestionPath)
    if (stillExists) {
      return
    }
    selectedSuggestionPath = suggestion.files[0]?.path ?? ''
  })

  $effect(() => {
    return () => {
      void closeActiveSession({ clearResult: false, suppressError: true })
    }
  })

  async function closeActiveSession(options: { clearResult: boolean; suppressError?: boolean }) {
    const activeSessionId = sessionId
    abortController?.abort()
    abortController = null
    pending = false
    sessionId = ''
    workspacePath = ''
    phase = ''
    phaseMessage = ''
    attempt = 0
    selectedSuggestionPath = ''
    if (options.clearResult) {
      result = null
    }
    if (!activeSessionId) {
      return
    }
    try {
      await closeSkillRefinementSession(activeSessionId)
    } catch (caughtError) {
      if (options.suppressError) {
        return
      }
      toastStore.error(
        caughtError instanceof Error
          ? caughtError.message
          : 'Failed to close skill refinement session.',
      )
    }
  }

  async function handleSend() {
    if (sendDisabled) {
      return
    }
    const message = prompt.trim()
    prompt = ''
    pending = true
    result = null
    dismissed = false
    phase = 'editing'
    phaseMessage = 'Preparing the draft bundle for Codex.'
    attempt = 0
    appliedBundleHash = ''

    const controller = new AbortController()
    abortController = controller

    try {
      await streamSkillRefinement(
        {
          projectId: projectId!,
          skillId: skillId!,
          message,
          providerId,
          files: files.map((file) => ({
            path: file.path,
            contentBase64: file.content_base64 ?? encodeUTF8Base64(file.content ?? ''),
            mediaType: file.media_type,
            isExecutable: file.is_executable,
          })),
        },
        {
          signal: controller.signal,
          onEvent: (event) => {
            switch (event.kind) {
              case 'session':
                sessionId = event.payload.sessionId
                workspacePath = event.payload.workspacePath
                return
              case 'status':
                phase = event.payload.phase
                phaseMessage = event.payload.message
                attempt = event.payload.attempt
                return
              case 'result':
                result = event.payload
                phase = event.payload.status
                phaseMessage =
                  event.payload.status === 'verified'
                    ? event.payload.transcriptSummary || 'Verification passed.'
                    : event.payload.failureReason || 'Verification did not pass.'
                attempt = event.payload.attempts
                return
              case 'error':
                phase = 'blocked'
                phaseMessage = event.payload.message
                toastStore.error(event.payload.message)
                return
            }
          },
        },
      )
    } catch (caughtError) {
      if (!(caughtError instanceof DOMException && caughtError.name === 'AbortError')) {
        toastStore.error(
          caughtError instanceof Error ? caughtError.message : 'Skill refinement failed.',
        )
        phase = 'blocked'
        phaseMessage =
          caughtError instanceof Error ? caughtError.message : 'Skill refinement failed.'
      }
    } finally {
      if (abortController === controller) {
        abortController = null
      }
      pending = false
    }
  }

  function handleApply() {
    if (!result || result.status !== 'verified') {
      return
    }
    onApplySuggestion?.(result.candidateFiles, selectedSuggestionPath || suggestion?.files[0]?.path)
    appliedBundleHash = result.candidateBundleHash ?? ''
  }

  function handleDismiss() {
    dismissed = true
  }

  async function handleProviderChange(nextProviderId: string) {
    if (nextProviderId === providerId) {
      return
    }
    providerId = nextProviderId
    await closeActiveSession({ clearResult: true })
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) {
      return
    }
    event.preventDefault()
    void handleSend()
  }

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

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-3 py-1">
    <div class="flex min-w-0 items-center gap-1.5">
      <span class="text-muted-foreground text-[11px] font-medium">Fix & verify</span>
      <EphemeralChatProviderSelect
        providers={refinementProviders}
        capability="skill_ai"
        {providerId}
        onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-6 p-0"
      aria-label="Reset refinement run"
      onclick={() => void closeActiveSession({ clearResult: true })}
      disabled={!sessionId && !result && !pending}
    >
      <Plus class="size-3" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-3 py-2">
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
      </div>

      {#if pending && !result && phase === 'editing'}
        <div
          class="flex items-center gap-2 rounded-md bg-sky-500/10 px-2.5 py-1.5 text-[11px] text-sky-300"
        >
          <LoaderCircle class="size-3 shrink-0 animate-spin" />
          Suggesting diff...
        </div>
      {/if}

      {#if result}
        <div class="space-y-3">
          {#if result.status === 'verified' && suggestion && preview && !dismissed}
            <SkillSuggestionCard
              {suggestion}
              selectedPath={selectedSuggestionPath}
              {preview}
              {suggestionAlreadyApplied}
              onSelectPath={(path) => (selectedSuggestionPath = path)}
              onApply={handleApply}
              onDismiss={handleDismiss}
            />
          {/if}

          {#if result.failureReason}
            <div class="rounded-lg border border-rose-500/30 bg-rose-500/8 p-3">
              <p class="text-[11px] font-medium tracking-[0.18em] text-rose-200 uppercase">
                Failure
              </p>
              <p class="mt-2 text-xs leading-5 whitespace-pre-wrap text-rose-50">
                {result.failureReason}
              </p>
            </div>
          {/if}

          {#if result.transcriptSummary}
            <div class="rounded-lg border border-white/8 bg-white/4 p-3">
              <div class="flex items-center gap-2">
                <Sparkles class="size-3.5 text-sky-200" />
                <p class="text-[11px] font-medium tracking-[0.18em] uppercase">
                  Transcript Summary
                </p>
              </div>
              <p class="mt-2 text-xs leading-5 whitespace-pre-wrap">{result.transcriptSummary}</p>
            </div>
          {/if}

          {#if result.commandOutputSummary}
            <div class="rounded-lg border border-white/8 bg-black/20 p-3">
              <p class="text-muted-foreground text-[11px] font-medium tracking-[0.18em] uppercase">
                Verification Output
              </p>
              <pre class="mt-2 font-mono text-[11px] leading-5 break-words whitespace-pre-wrap">
{result.commandOutputSummary}</pre>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </ScrollArea>

  <div class="border-border border-t px-3 py-2">
    <div class="space-y-2">
      <Textarea
        bind:value={prompt}
        rows={4}
        class="min-h-[88px] resize-none text-xs"
        placeholder="Describe what Codex should improve and verify for this draft bundle…"
        disabled={!projectId || !skillId || !providerId || pending}
        onkeydown={handlePromptKeydown}
      />
      <Button
        class="h-8 w-full gap-2 text-xs"
        onclick={() => void handleSend()}
        disabled={sendDisabled}
      >
        <ShieldCheck class="size-3.5" />
        {pending ? 'Running verification…' : 'Fix and verify'}
      </Button>
    </div>
  </div>
</div>
