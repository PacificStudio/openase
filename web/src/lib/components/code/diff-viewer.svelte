<script lang="ts">
  import { cn } from '$lib/utils'

  let {
    diff = '',
    sourceContent = '',
    class: className = '',
  }: {
    /** Raw unified diff string */
    diff?: string
    /** Full file content (new/current version). When provided, the viewer shows
     *  the entire file with diff hunks highlighted inline instead of only the
     *  raw diff output. */
    sourceContent?: string
    class?: string
  } = $props()

  type LineKind = 'add' | 'del' | 'context' | 'hunk-sep'

  interface AnnotatedLine {
    text: string
    kind: LineKind
    /** Line number in the *new* file. null for deleted lines. */
    newLineNo: number | null
    /** Line number in the *old* file. null for added lines. */
    oldLineNo: number | null
  }

  interface Hunk {
    oldStart: number
    oldCount: number
    newStart: number
    newCount: number
    lines: { text: string; kind: 'add' | 'del' | 'context' }[]
  }

  function parseHunks(raw: string): Hunk[] {
    const hunks: Hunk[] = []
    const allLines = raw.split('\n')
    let current: Hunk | null = null

    for (const line of allLines) {
      const hunkMatch = line.match(/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/)
      if (hunkMatch) {
        current = {
          oldStart: parseInt(hunkMatch[1], 10),
          oldCount: hunkMatch[2] != null ? parseInt(hunkMatch[2], 10) : 1,
          newStart: parseInt(hunkMatch[3], 10),
          newCount: hunkMatch[4] != null ? parseInt(hunkMatch[4], 10) : 1,
          lines: [],
        }
        hunks.push(current)
        continue
      }

      if (!current) continue

      // Skip diff file headers
      if (
        line.startsWith('diff --git') ||
        line.startsWith('index ') ||
        line.startsWith('---') ||
        line.startsWith('+++')
      ) {
        continue
      }

      if (line.startsWith('+')) {
        current.lines.push({ text: line.slice(1), kind: 'add' })
      } else if (line.startsWith('-')) {
        current.lines.push({ text: line.slice(1), kind: 'del' })
      } else if (line.startsWith(' ') || line === '') {
        // Context line — the leading space is part of the diff format
        current.lines.push({ text: line.startsWith(' ') ? line.slice(1) : line, kind: 'context' })
      }
    }
    return hunks
  }

  /**
   * Merge the full file content with parsed hunks to produce an annotated line
   * list that covers the **entire** file, with additions/deletions highlighted.
   */
  function mergeSourceWithDiff(source: string, hunks: Hunk[]): AnnotatedLine[] {
    const sourceLines = source.split('\n')
    // Remove trailing empty line that split produces for files ending with \n
    if (sourceLines.length > 0 && sourceLines[sourceLines.length - 1] === '') {
      sourceLines.pop()
    }

    const result: AnnotatedLine[] = []
    let srcIdx = 0 // 0-based index into sourceLines (tracks new-file line number)

    for (const hunk of hunks) {
      const hunkNewStart = hunk.newStart - 1 // convert to 0-based

      // Emit unchanged lines before this hunk
      while (srcIdx < hunkNewStart && srcIdx < sourceLines.length) {
        result.push({
          text: sourceLines[srcIdx],
          kind: 'context',
          newLineNo: srcIdx + 1,
          oldLineNo: null, // we don't track old line numbers for gap lines
        })
        srcIdx++
      }

      // Separator between file context and hunk
      if (result.length > 0) {
        result.push({
          text: `@@ -${hunk.oldStart},${hunk.oldCount} +${hunk.newStart},${hunk.newCount} @@`,
          kind: 'hunk-sep',
          newLineNo: null,
          oldLineNo: null,
        })
      }

      // Emit hunk lines
      let newLine = hunk.newStart
      let oldLine = hunk.oldStart
      for (const hLine of hunk.lines) {
        if (hLine.kind === 'del') {
          result.push({ text: hLine.text, kind: 'del', newLineNo: null, oldLineNo: oldLine })
          oldLine++
        } else if (hLine.kind === 'add') {
          result.push({ text: hLine.text, kind: 'add', newLineNo: newLine, oldLineNo: null })
          newLine++
          srcIdx++ // advance source pointer past this added line
        } else {
          result.push({ text: hLine.text, kind: 'context', newLineNo: newLine, oldLineNo: oldLine })
          newLine++
          oldLine++
          srcIdx++
        }
      }
    }

    // Emit remaining lines after the last hunk
    while (srcIdx < sourceLines.length) {
      result.push({
        text: sourceLines[srcIdx],
        kind: 'context',
        newLineNo: srcIdx + 1,
        oldLineNo: null,
      })
      srcIdx++
    }

    return result
  }

  /**
   * Fallback: parse the raw diff without source content (original behavior).
   */
  function diffOnlyLines(raw: string): AnnotatedLine[] {
    return raw.split('\n').map((line, i) => {
      let kind: LineKind = 'context'
      if (line.startsWith('@@')) kind = 'hunk-sep'
      else if (line.startsWith('+')) kind = 'add'
      else if (line.startsWith('-')) kind = 'del'
      return { text: line, kind, newLineNo: i + 1, oldLineNo: null }
    })
  }

  const annotated = $derived.by(() => {
    const raw = diff ?? ''
    if (!raw) return []
    if (sourceContent) {
      const hunks = parseHunks(raw)
      if (hunks.length > 0) {
        return mergeSourceWithDiff(sourceContent, hunks)
      }
    }
    return diffOnlyLines(raw)
  })
</script>

<div class={cn('diff-viewer min-h-0 overflow-auto font-mono text-[13px] leading-6', className)}>
  <div class="px-0 py-2">
    {#each annotated as line}
      <div
        class={cn(
          'min-h-6 px-4 whitespace-pre-wrap',
          line.kind === 'add' && 'bg-emerald-500/10 text-emerald-800 dark:text-emerald-300',
          line.kind === 'del' && 'bg-rose-500/10 text-rose-800 dark:text-rose-300',
          line.kind === 'hunk-sep' && 'bg-sky-500/8 text-[11px] text-sky-600 dark:text-sky-400',
        )}
      >
        <span class="mr-3 inline-block w-8 text-right text-[11px] opacity-40 select-none"
          >{line.newLineNo ?? ''}</span
        >{line.text}
      </div>
    {/each}
  </div>
</div>
