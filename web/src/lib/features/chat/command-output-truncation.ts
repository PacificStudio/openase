export function truncateInline(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength - 3)}...`
}

export function countOutputLines(text: string): number {
  return text.split('\n').length
}

export function truncateOutput(
  text: string,
  headLines = 5,
  tailLines = 5,
): { head: string; omitted: number; tail: string } | null {
  const lines = text.split('\n')
  const total = lines.length
  if (total <= headLines + tailLines) return null

  const omitted = total - headLines - tailLines
  return {
    head: lines.slice(0, headLines).join('\n'),
    omitted,
    tail: lines.slice(total - tailLines).join('\n'),
  }
}
