import type { TicketRunTranscriptBlock } from './types'

/**
 * A content block rendered prominently (assistant_message, tool_call, terminal_output, interrupt,
 * result, task_status, diff).
 */
export type ContentItem = {
  type: 'content'
  block: TicketRunTranscriptBlock
}

/**
 * A group of consecutive noise blocks (phase, step) rendered as a
 * single collapsible one-liner, following the same pattern as Project AI's
 * operation groups.
 */
export type NoiseGroup = {
  type: 'noise_group'
  id: string
  blocks: TicketRunTranscriptBlock[]
  summary: string
  detail: string
}

export type TranscriptDisplayItem = ContentItem | NoiseGroup

function isNoiseBlock(block: TicketRunTranscriptBlock): boolean {
  if (block.kind === 'phase') return true
  return false
}

function buildNoiseSummary(blocks: TicketRunTranscriptBlock[]): {
  summary: string
  detail: string
} {
  const phases = blocks.filter((b) => b.kind === 'phase') as Extract<
    TicketRunTranscriptBlock,
    { kind: 'phase' }
  >[]
  const steps = blocks.filter((b) => b.kind === 'step') as Extract<
    TicketRunTranscriptBlock,
    { kind: 'step' }
  >[]
  const tools = blocks.filter((b) => b.kind === 'tool_call') as Extract<
    TicketRunTranscriptBlock,
    { kind: 'tool_call' }
  >[]

  // Single block — use its own summary
  if (blocks.length === 1) {
    const b = blocks[0]
    if (b.kind === 'phase') return { summary: `Phase: ${b.phase}`, detail: b.summary }
    if (b.kind === 'step') return { summary: `Step: ${b.stepStatus}`, detail: b.summary }
    if (b.kind === 'tool_call') return { summary: b.toolName, detail: b.summary ?? '' }
    return { summary: 'System event', detail: '' }
  }

  const parts: string[] = []
  if (phases.length > 0) parts.push(`${phases.length} phase(s)`)
  if (steps.length > 0) parts.push(`${steps.length} step(s)`)
  if (tools.length > 0) parts.push(`${tools.length} tool call(s)`)

  // Use latest phase as headline if available
  const latestPhase = phases.at(-1)
  const summary = latestPhase ? `Phase: ${latestPhase.phase}` : `${blocks.length} operations`

  return { summary, detail: parts.join(', ') }
}

/**
 * Merge consecutive assistant_message blocks into one (like Project AI's delta
 * merging) and group noise blocks into collapsible one-liners.
 */
export function groupRunTranscriptBlocks(
  blocks: TicketRunTranscriptBlock[],
): TranscriptDisplayItem[] {
  const items: TranscriptDisplayItem[] = []
  let noiseGroup: TicketRunTranscriptBlock[] = []

  function flushNoise() {
    if (noiseGroup.length === 0) return
    const { summary, detail } = buildNoiseSummary(noiseGroup)
    items.push({
      type: 'noise_group',
      id: `noise-${noiseGroup[0].id}`,
      blocks: noiseGroup,
      summary,
      detail,
    })
    noiseGroup = []
  }

  // First pass: merge consecutive assistant_message blocks
  const merged: TicketRunTranscriptBlock[] = []
  for (const block of blocks) {
    if (block.kind === 'assistant_message') {
      const prev = merged.at(-1)
      if (prev && prev.kind === 'assistant_message') {
        // Merge into previous
        merged[merged.length - 1] = {
          ...prev,
          text: prev.text + block.text,
          streaming: block.streaming,
        }
        continue
      }
    }
    merged.push(block)
  }

  // Second pass: group noise vs content
  for (const block of merged) {
    if (isNoiseBlock(block)) {
      noiseGroup.push(block)
    } else {
      flushNoise()
      items.push({ type: 'content', block })
    }
  }

  flushNoise()
  return items
}
