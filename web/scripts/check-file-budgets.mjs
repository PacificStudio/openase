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
      'src/lib/features/agents/components/agent-drawer.svelte':
        'Agent drawer is temporarily oversized while provider and runtime controls remain in one panel.',
      'src/lib/features/dashboard/components/org-dashboard.svelte':
        'Dashboard summary and advisor layout remain colocated pending dashboard panel extraction.',
      'src/lib/features/onboarding/components/onboarding-panel.svelte':
        'Onboarding panel still coordinates the full multi-step flow until onboarding controller extraction lands.',
      'src/lib/features/onboarding/components/step-repo.svelte':
        'Onboarding repo step keeps create and link flows together until onboarding step extraction lands.',
      'src/lib/features/project-updates/components/project-update-thread-card.svelte':
        'Project update thread card still keeps inline edit and reply flows together until card sections are extracted.',
      'src/lib/features/settings/components/workflow-scheduled-job-cron-picker.svelte':
        'Cron picker keeps presets and field editor together until scheduled job form extraction lands.',
      'src/lib/features/settings/components/workflow-scheduled-jobs-panel.svelte':
        'Scheduled jobs panel still coordinates list, editor, and summary sections until panel extraction lands.',
      'src/lib/features/skills/components/skill-editor-page.svelte':
        'Skill editor page still hosts save/binding orchestration until controller extraction lands.',
      'src/lib/features/skills/components/skills-page.svelte':
        'Skills page still coordinates browser, selection, and editor loading until page controller extraction lands.',
      'src/lib/features/ticket-detail/components/ticket-drawer.svelte':
        'Ticket drawer still coordinates detail shell, panes, and runtime state until drawer extraction lands.',
      'src/lib/features/tickets/components/new-ticket-dialog.svelte':
        'Ticket creation dialog still hosts staged form orchestration until dialog sections are extracted.',
      'src/lib/features/tickets/components/tickets-page.svelte':
        'Tickets page still owns board loading and stream orchestration until board controller extraction lands.',
      'src/lib/features/workflows/components/workflows-page.svelte':
        'Workflows page still coordinates editor, history, and skill bindings until controller extraction lands.',
    },
  },
  {
    name: 'Feature modules',
    match: (filePath) => /^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/.test(filePath),
    softLimit: 200,
    hardLimit: 300,
    allowlist: {
      'src/lib/features/agents/model.test.ts':
        'Agent provider normalization coverage remains in one test module until scenario helpers are extracted.',
      'src/lib/features/board/components/board-page-controls.test.ts':
        'Board controls regression coverage remains consolidated while the control matrix is still evolving.',
      'src/lib/features/board/components/board-page-runtime-and-drawer.test.ts':
        'Board runtime stream and drawer regression coverage remain bundled while ticket board helpers are still being split out.',
      'src/lib/features/chat/project-conversation-controller.svelte.ts':
        'Project conversation controller remains centralized while multi-tab restore and per-tab runtime flows continue to settle.',
      'src/lib/features/chat/project-conversation-controller.test.ts':
        'Project conversation controller regression coverage remains bundled while multi-tab conversation scenarios are still expanding.',
      'src/lib/features/skills/components/skill-bundle-editor.ts':
        'Skill bundle draft helpers remain centralized until tree and path utilities are split out.',
      'src/lib/features/workflows/components/harness-ai-sidebar-streaming.test.ts':
        'Harness AI long-stream regression coverage stays in one integration-style test while SSE helpers remain inline.',
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
