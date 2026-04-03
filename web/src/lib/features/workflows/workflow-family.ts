import type { WorkflowClassification, WorkflowFamily } from './types'

const knownWorkflowFamilies: WorkflowFamily[] = [
  'planning',
  'dispatcher',
  'coding',
  'review',
  'test',
  'docs',
  'deploy',
  'security',
  'harness',
  'environment',
  'research',
  'reporting',
  'unknown',
]

export const workflowFamilyColors: Record<WorkflowFamily, string> = {
  planning: 'bg-sky-500/15 text-sky-400 border-sky-500/20',
  dispatcher: 'bg-neutral-500/15 text-neutral-300 border-neutral-500/20',
  coding: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  review: 'bg-fuchsia-500/15 text-fuchsia-400 border-fuchsia-500/20',
  test: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20',
  docs: 'bg-violet-500/15 text-violet-400 border-violet-500/20',
  deploy: 'bg-amber-500/15 text-amber-400 border-amber-500/20',
  security: 'bg-rose-500/15 text-rose-400 border-rose-500/20',
  harness: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/20',
  environment: 'bg-teal-500/15 text-teal-400 border-teal-500/20',
  research: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/20',
  reporting: 'bg-orange-500/15 text-orange-400 border-orange-500/20',
  unknown: 'bg-neutral-500/15 text-neutral-400 border-neutral-500/20',
}

export const workflowFamilyIcons: Record<WorkflowFamily, string> = {
  planning: '🧭',
  dispatcher: '🗂️',
  coding: '💻',
  review: '🧾',
  test: '🧪',
  docs: '📝',
  deploy: '🚀',
  security: '🔒',
  harness: '🔧',
  environment: '🛠️',
  research: '🔬',
  reporting: '📄',
  unknown: '⚙️',
}

export const workflowFamilyDescriptions: Record<WorkflowFamily, string> = {
  planning: 'Planning and scope definition',
  dispatcher: 'Ticket routing and lane assignment',
  coding: 'Implementation and delivery',
  review: 'Review and approval',
  test: 'QA and verification',
  docs: 'Documentation and guides',
  deploy: 'Release and rollout',
  security: 'Security review and hardening',
  harness: 'Harness and prompt optimization',
  environment: 'Environment setup and repair',
  research: 'Research and experiments',
  reporting: 'Reports and writeups',
  unknown: 'Unclassified workflow family',
}

export function normalizeWorkflowFamily(family: string): WorkflowFamily {
  return knownWorkflowFamilies.includes(family as WorkflowFamily)
    ? (family as WorkflowFamily)
    : 'unknown'
}

export function normalizeWorkflowClassification(
  classification:
    | Partial<WorkflowClassification>
    | {
        family?: string
        confidence?: number
        reasons?: unknown[]
      }
    | null
    | undefined,
  fallbackFamily: string,
): WorkflowClassification {
  const reasons = (Array.isArray(classification?.reasons) ? classification.reasons : []).reduce<
    string[]
  >((items, value) => {
    if (typeof value === 'string') {
      items.push(value)
    }
    return items
  }, [])

  return {
    family: normalizeWorkflowFamily(classification?.family ?? fallbackFamily),
    confidence:
      typeof classification?.confidence === 'number' && Number.isFinite(classification.confidence)
        ? classification.confidence
        : 0,
    reasons,
  }
}
