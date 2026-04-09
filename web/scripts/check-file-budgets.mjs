import fs from 'node:fs'
import path from 'node:path'

const repoRoot = process.cwd()
const sourceRoot = path.join(repoRoot, 'src')

const budgetRules = [
  {
    name: 'Route pages',
    match: (filePath) =>
      /^src\/routes\/.+\/\+page\.svelte$|^src\/routes\/\+page\.svelte$/.test(filePath),
    softLimit: 150,
    hardLimit: 250,
  },
  {
    name: 'Route layouts',
    match: (filePath) =>
      /^src\/routes\/.+\/\+layout\.svelte$|^src\/routes\/\+layout\.svelte$/.test(filePath),
    softLimit: 180,
    hardLimit: 300,
  },
  {
    name: 'Feature components',
    match: (filePath) => /^src\/lib\/features\/.+\.svelte$/.test(filePath),
    softLimit: 200,
    hardLimit: 300,
    allowlist: {
      'src/lib/features/app-shell/components/project-shell.svelte':
        'Project shell still coordinates top bar, sidebar, app-context refresh, and overlays while shell controller extraction continues.',
      'src/lib/features/workflows/components/workflow-creation-dialog.svelte':
        'Workflow creation still combines label, status binding, agent binding, and hook editing while the creation form controller is being extracted.',
      'src/lib/features/settings/components/general-settings.svelte':
        'General settings currently combines run summary prompt builder and archive controls while those panels remain in a single form surface.',
    },
  },
  {
    name: 'Feature modules',
    match: (filePath) => /^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/.test(filePath),
    softLimit: 200,
    hardLimit: 300,
    allowlist: {
      'src/lib/features/board/components/board-page-controls.test.ts':
        'Board controls regression coverage remains consolidated while the control matrix is still evolving.',
      'src/lib/features/board/components/board-page-runtime-and-drawer.test.ts':
        'Board runtime stream and drawer regression coverage remain bundled while ticket board helpers are still being split out.',
      'src/lib/features/chat/project-conversation-controller.test.ts':
        'Project conversation controller regression coverage remains bundled while multi-tab conversation scenarios are still expanding.',
      'src/lib/features/chat/project-conversation-controller-restore.test.ts':
        'Project conversation restore regression coverage remains bundled while session restore and workspace diff helpers are still being extracted.',
      'src/lib/features/chat/project-conversation-controller.svelte.ts':
        'Project conversation controller still coordinates tab selection, provider sync, and delegated operations while the remaining controller slices continue to move out.',
      'src/lib/features/chat/project-conversation-panel.test.ts':
        'Project conversation panel regression coverage remains bundled while transcript, restore, and action-surface scenarios continue to share setup.',
      'src/lib/features/chat/project-conversation-panel-tabs.test.ts':
        'Project conversation tab behavior coverage remains bundled while queued-turn and interrupt scenarios still share the same panel harness.',
      'src/lib/features/dashboard/components/org-dashboard-controller.svelte.ts':
        'Organization dashboard controller still owns event-driven refresh orchestration while dashboard loading helpers continue to move out.',
      'src/lib/features/providers/rate-limit.ts':
        'Provider rate-limit summarization still centralizes Claude, Codex, and Gemini snapshot shaping while adapter-specific presenters remain in one module.',
      'src/lib/features/skills/components/skill-editor-page.test.ts':
        'Skill editor page interaction coverage remains bundled while keyboard, draft, and binding scenarios continue to expand alongside the editor controller.',
      'src/lib/features/ticket-detail/run-transcript.test.ts':
        'Run transcript reducer regression coverage remains consolidated while lifecycle, summary, and trace fixtures continue to share setup.',
      'src/lib/features/workflows/components/workflows-page.test.ts':
        'Workflow page interaction coverage remains bundled until test helper extraction lands.',
    },
  },
  {
    name: 'Layout components',
    match: (filePath) => /^src\/lib\/components\/layout\/.+\.svelte$/.test(filePath),
    softLimit: 200,
    hardLimit: 300,
  },
  {
    name: 'UI primitives',
    match: (filePath) => /^src\/lib\/components\/ui\/.+\.svelte$/.test(filePath),
    softLimit: 150,
    hardLimit: 250,
  },
]

const files = walkFiles(sourceRoot)
const warnings = []
const failures = []
const waived = []

for (const absoluteFile of files) {
  const relativeFile = toRepoPath(absoluteFile)
  const budget = budgetRules.find((rule) => rule.match(relativeFile))
  if (!budget) {
    continue
  }

  const lineCount = fs.readFileSync(absoluteFile, 'utf8').split(/\r?\n/).length
  if (lineCount > budget.hardLimit) {
    const waiverReason = budget.allowlist?.[relativeFile]
    if (waiverReason) {
      waived.push({ file: relativeFile, lineCount, budget, waiverReason })
    } else {
      failures.push({ file: relativeFile, lineCount, budget })
    }
    continue
  }

  if (lineCount > budget.softLimit) {
    warnings.push({ file: relativeFile, lineCount, budget })
  }
}

if (waived.length > 0) {
  console.warn('Legacy file budget waivers:')
  for (const item of waived) {
    console.warn(`  - ${item.file}: ${item.lineCount} lines (hard limit ${item.budget.hardLimit})`)
    console.warn(`    waiver: ${item.waiverReason}`)
  }
  console.warn('')
}

if (warnings.length > 0) {
  console.warn('Soft budget warnings:')
  for (const item of warnings) {
    console.warn(`  - ${item.file}: ${item.lineCount} lines (soft limit ${item.budget.softLimit})`)
  }
  console.warn('')
}

if (failures.length > 0) {
  console.error('File budget violations:')
  for (const item of failures) {
    console.error(`  - ${item.file}: ${item.lineCount} lines (hard limit ${item.budget.hardLimit})`)
  }
  process.exit(1)
}

process.stdout.write(`File budgets passed for ${files.length} source files.\n`)

function walkFiles(directoryPath) {
  if (!fs.existsSync(directoryPath)) {
    return []
  }

  return fs.readdirSync(directoryPath, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directoryPath, entry.name)
    if (entry.isDirectory()) {
      return walkFiles(entryPath)
    }
    return entry.isFile() ? [entryPath] : []
  })
}

function toRepoPath(filePath) {
  return path.relative(repoRoot, filePath).split(path.sep).join('/')
}
