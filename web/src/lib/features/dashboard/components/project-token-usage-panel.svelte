<script lang="ts">
  import {
    loadProjectTokenUsage,
    emptyTokenUsageAnalytics,
  } from '$lib/features/dashboard/token-usage'
  import { appStore } from '$lib/stores/app.svelte'
  import TokenUsageAnalyticsPanel from './token-usage-analytics-panel.svelte'
  import type { TokenUsageAnalytics, TokenUsageRange } from '../types'

  let analyticsLoading = $state(false)
  let selectedUsageRange = $state<TokenUsageRange>(30)
  let tokenUsage = $state<TokenUsageAnalytics>(emptyTokenUsageAnalytics(30))

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const refreshKey = `${projectId ?? ''}:${selectedUsageRange}:${appStore.appContextFetchedAt}`
    void refreshKey

    if (!projectId) {
      tokenUsage = emptyTokenUsageAnalytics(selectedUsageRange)
      return
    }

    let cancelled = false
    const abortController = new AbortController()

    const load = async () => {
      analyticsLoading = true

      try {
        const analytics = await loadProjectTokenUsage(projectId, selectedUsageRange, {
          signal: abortController.signal,
        })
        if (cancelled) return
        tokenUsage = analytics
      } catch {
        if (cancelled || abortController.signal.aborted) return
        tokenUsage = emptyTokenUsageAnalytics(selectedUsageRange)
      } finally {
        if (!cancelled) analyticsLoading = false
      }
    }

    void load()
    return () => {
      cancelled = true
      abortController.abort()
    }
  })
</script>

<TokenUsageAnalyticsPanel
  analytics={tokenUsage}
  selectedRange={selectedUsageRange}
  loading={analyticsLoading}
  onSelectRange={(range) => {
    selectedUsageRange = range
  }}
/>
