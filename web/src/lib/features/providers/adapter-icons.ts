import claudeCodeIcon from '$lib/assets/claudecode.svg'
import codexIcon from '$lib/assets/codex.svg'
import geminiIcon from '$lib/assets/gemini.svg'

const adapterIcons: Record<string, string> = {
  'claude-code-cli': claudeCodeIcon,
  'codex-app-server': codexIcon,
  'gemini-cli': geminiIcon,
}

const adapterNames: Record<string, string> = {
  'claude-code-cli': 'Claude Code',
  'codex-app-server': 'Codex',
  'gemini-cli': 'Gemini',
}

export function adapterIconPath(adapterType: string): string {
  return adapterIcons[adapterType] ?? ''
}

export function adapterDisplayName(adapterType: string): string {
  return adapterNames[adapterType] ?? adapterType
}

export function availabilityDotColor(available: boolean | undefined): string {
  if (available === true) return 'bg-emerald-500'
  if (available === false) return 'bg-rose-500'
  return 'bg-slate-400'
}
