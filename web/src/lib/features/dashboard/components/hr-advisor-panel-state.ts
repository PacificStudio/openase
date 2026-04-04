import type { HRAdvisorRecommendation } from '$lib/api/contracts'
import type { HRAdvisorSnapshot } from '../types'

export type PrioritySectionKey = 'high' | 'medium' | 'low' | 'other'

export type PrioritySectionMeta = {
  key: PrioritySectionKey
  label: string
  accentClass: string
}

const deferredStoragePrefix = 'openase.dashboard.hrAdvisor.deferred.'

export const prioritySectionsMeta: PrioritySectionMeta[] = [
  {
    key: 'high',
    label: '高优先级',
    accentClass: 'border-rose-500/20 bg-rose-500/5',
  },
  {
    key: 'medium',
    label: '中优先级',
    accentClass: 'border-amber-500/20 bg-amber-500/5',
  },
  {
    key: 'low',
    label: '低优先级',
    accentClass: 'border-sky-500/20 bg-sky-500/5',
  },
  {
    key: 'other',
    label: '其他建议',
    accentClass: 'border-border bg-muted/20',
  },
]

export function recommendationKey(
  recommendation: Pick<HRAdvisorRecommendation, 'role_slug' | 'suggested_workflow_name'>,
) {
  return `${recommendation.role_slug}:${recommendation.suggested_workflow_name}`
}

export function toPrioritySectionKey(priority: string): PrioritySectionKey {
  if (priority === 'high' || priority === 'medium' || priority === 'low') {
    return priority
  }
  return 'other'
}

export function activationStatusText(recommendation: HRAdvisorRecommendation) {
  if (recommendation.activation_ready) {
    return `可立即创建 ${recommendation.suggested_workflow_name} workflow。`
  }

  const workflowName = recommendation.active_workflow_name || recommendation.suggested_workflow_name
  return `已通过 ${workflowName} 激活。`
}

export function loadDeferredRecommendationKeys(projectId: string) {
  if (typeof window === 'undefined') {
    return []
  }

  const raw = window.localStorage.getItem(storageKey(projectId))
  if (!raw) {
    return []
  }

  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed)
      ? parsed.filter((item): item is string => typeof item === 'string')
      : []
  } catch {
    return []
  }
}

export function persistDeferredRecommendationKeys(projectId: string, nextKeys: string[]) {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(storageKey(projectId), JSON.stringify(nextKeys))
  }
  return nextKeys
}

export function applyActivatedRecommendation(
  currentAdvisor: HRAdvisorSnapshot,
  recommendation: HRAdvisorRecommendation,
  activeWorkflowName: string,
) {
  const key = recommendationKey(recommendation)

  return {
    ...currentAdvisor,
    recommendations: currentAdvisor.recommendations.map((item) =>
      recommendationKey(item) === key
        ? {
            ...item,
            activation_ready: false,
            active_workflow_name: activeWorkflowName,
          }
        : item,
    ),
  }
}

function storageKey(projectId: string) {
  return `${deferredStoragePrefix}${projectId}`
}
