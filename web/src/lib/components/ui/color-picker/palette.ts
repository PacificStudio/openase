/**
 * Curated status color palette.
 *
 * - `quickPicks`: 5 high-distinctiveness colors for one-click status assignment.
 * - `expanded`: 40 colors (5 rows × 8 columns) organized by hue family,
 *   suitable for scanning and quick selection.
 *
 * All values are lowercase 6-digit hex (#rrggbb).
 */

/** Five default quick-pick colors — high contrast, semantically distinct. */
export const quickPicks: string[] = [
  '#3b82f6', // blue — info / default / in-progress
  '#22c55e', // green — success / done / active
  '#f59e0b', // amber — warning / review / pending
  '#ef4444', // red — error / blocked / urgent
  '#8b5cf6', // violet — special / custom / deferred
]

/**
 * 40 curated preset colors arranged as 5 rows × 8 columns.
 * Each row follows a hue family from cool to warm, with subtle
 * lightness/saturation variation across columns.
 */
export const expandedPalette: string[][] = [
  // Row 1: Blues & Cyans
  ['#1e3a5f', '#1e40af', '#2563eb', '#3b82f6', '#60a5fa', '#06b6d4', '#22d3ee', '#67e8f9'],
  // Row 2: Greens & Teals
  ['#15803d', '#16a34a', '#22c55e', '#4ade80', '#86efac', '#0d9488', '#14b8a6', '#5eead4'],
  // Row 3: Yellows & Oranges
  ['#92400e', '#b45309', '#d97706', '#f59e0b', '#fbbf24', '#ea580c', '#f97316', '#fb923c'],
  // Row 4: Reds & Pinks
  ['#991b1b', '#dc2626', '#ef4444', '#f87171', '#fca5a5', '#be185d', '#ec4899', '#f9a8d4'],
  // Row 5: Purples & Neutrals
  ['#4c1d95', '#6d28d9', '#8b5cf6', '#a78bfa', '#c4b5fd', '#475569', '#64748b', '#94a3b8'],
]

/** Flat list of all 40 expanded colors for lookup. */
export const allPresets: string[] = expandedPalette.flat()

/** All selectable preset colors (quick picks + expanded). */
export const allColors: string[] = [...new Set([...quickPicks, ...allPresets])]

/** Check if a hex value is one of the curated presets. */
export function isPresetColor(hex: string): boolean {
  return allColors.includes(hex.toLowerCase())
}
