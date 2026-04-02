<script lang="ts">
  import { tick } from 'svelte'
  import type { StreamConnectionState } from '$lib/api/sse'
  import TicketRunHistoryAttempt from './ticket-run-history-attempt.svelte'
  import { FINISHED_RUN_STATUSES, renderKeyForBlocks } from './ticket-run-history-panel-view-model'
  import type { TicketDetail, TicketRun, TicketRunTranscriptBlock } from '../types'

  const autoFollowThresholdPx = 80

  let {
    ticket,
    runs = [],
    currentRun = null,
    blocks = [],
    loadingRunId = null,
    runStreamState = 'idle',
    recoveringRunTranscript = false,
    resumingRetry = false,
    onSelectRun,
    onResumeRetry,
  }: {
    ticket: TicketDetail
    runs?: TicketRun[]
    currentRun?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
    loadingRunId?: string | null
    runStreamState?: StreamConnectionState
    recoveringRunTranscript?: boolean
    resumingRetry?: boolean
    onSelectRun?: (runId: string) => Promise<void> | void
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  let sectionEl = $state<HTMLElement | null>(null)
  let expandedOutputIds = $state<string[]>([])
  let expandedNoiseGroups = $state(new Set<string>())
  let autoFollow = $state(true)
  let showJumpToLive = $state(false)
  let previousSelectionKey = ''
  let previousRenderKey = ''

  /** Walk up the DOM to find the nearest scrollable ancestor. */
  function getScrollViewport(): HTMLElement | null {
    let el = sectionEl?.parentElement ?? null
    while (el) {
      const style = getComputedStyle(el)
      if (
        (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
        el.scrollHeight > el.clientHeight
      ) {
        return el
      }
      el = el.parentElement
    }
    return null
  }

  const latestRun = $derived(runs[0] ?? null)
  const chronologicalRuns = $derived([...runs].reverse())
  const liveSelected = $derived(currentRun != null && latestRun?.id === currentRun.id)

  const canResumeRetry = $derived(
    ticket.retryPaused &&
      ticket.pauseReason === 'repeated_stalls' &&
      latestRun?.id === currentRun?.id,
  )

  function isOutputExpanded(blockId: string) {
    return expandedOutputIds.includes(blockId)
  }

  async function toggleOutputExpansion(blockId: string) {
    const viewport = getScrollViewport()
    const bottomOffset = viewport
      ? Math.max(viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop, 0)
      : 0

    expandedOutputIds = isOutputExpanded(blockId)
      ? expandedOutputIds.filter((item) => item !== blockId)
      : [...expandedOutputIds, blockId]

    await tick()

    if (!viewport) return
    if (liveSelected && autoFollow) {
      jumpToBottom()
      return
    }
    viewport.scrollTop = Math.max(viewport.scrollHeight - viewport.clientHeight - bottomOffset, 0)
  }

  function toggleNoiseGroup(groupId: string) {
    const next = new Set(expandedNoiseGroups)
    if (next.has(groupId)) {
      next.delete(groupId)
    } else {
      next.add(groupId)
    }
    expandedNoiseGroups = next
  }

  function selectRun(runId: string) {
    if (currentRun?.id === runId) return
    expandedOutputIds = []
    expandedNoiseGroups = new Set()
    void onSelectRun?.(runId)
  }

  function jumpToBottom() {
    const viewport = getScrollViewport()
    if (!viewport) return
    viewport.scrollTop = viewport.scrollHeight
  }

  async function jumpToLive() {
    autoFollow = true
    showJumpToLive = false
    await tick()
    jumpToBottom()
  }

  function isNearBottom() {
    const viewport = getScrollViewport()
    if (!viewport) return true
    return (
      viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop <= autoFollowThresholdPx
    )
  }

  function handleScroll() {
    if (!liveSelected) {
      showJumpToLive = false
      return
    }
    autoFollow = isNearBottom()
    showJumpToLive = !autoFollow
  }

  $effect(() => {
    if (!sectionEl) return
    const viewport = getScrollViewport()
    if (!viewport) return

    viewport.addEventListener('scroll', handleScroll)
    return () => viewport.removeEventListener('scroll', handleScroll)
  })

  $effect(() => {
    const selectionKey = `${currentRun?.id ?? 'none'}:${latestRun?.id ?? 'none'}`
    if (selectionKey === previousSelectionKey) return
    previousSelectionKey = selectionKey
    autoFollow = liveSelected
    showJumpToLive = false
    expandedOutputIds = []
  })

  $effect(() => {
    const shouldStickToBottom = liveSelected && autoFollow && isNearBottom()
    const nextRenderKey = `${currentRun?.id ?? 'none'}:${blocks.length}:${renderKeyForBlocks(blocks)}:${runStreamState}:${recoveringRunTranscript}`
    if (nextRenderKey === previousRenderKey) return
    previousRenderKey = nextRenderKey
    void tick().then(() => {
      if (shouldStickToBottom) {
        jumpToBottom()
      } else if (liveSelected) {
        showJumpToLive = !isNearBottom()
      }
    })
  })
</script>

{#if runs.length === 0}
  <section class="px-4 py-6">
    <p class="text-muted-foreground text-xs">No runs yet.</p>
  </section>
{:else}
  <section bind:this={sectionEl}>
    {#each chronologicalRuns as run (run.id)}
      {@const selected = currentRun?.id === run.id}
      {@const live = latestRun?.id === run.id && !FINISHED_RUN_STATUSES.has(run.status)}
      {@const loading = loadingRunId === run.id}
      <TicketRunHistoryAttempt
        {ticket}
        {run}
        {selected}
        {live}
        {loading}
        blocks={selected ? blocks : []}
        {runStreamState}
        {recoveringRunTranscript}
        {liveSelected}
        {expandedOutputIds}
        {expandedNoiseGroups}
        {canResumeRetry}
        {resumingRetry}
        {showJumpToLive}
        onSelectRun={selectRun}
        onToggleOutput={toggleOutputExpansion}
        onToggleNoiseGroup={toggleNoiseGroup}
        {onResumeRetry}
        onJumpToLive={jumpToLive}
      />
    {/each}
  </section>
{/if}
