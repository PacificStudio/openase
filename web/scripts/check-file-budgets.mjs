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
      'src/lib/features/dashboard/components/org-dashboard.svelte':
        'Dashboard shell still mixes overview KPIs, token analytics, and advisor lane composition until each panel owns its own loading boundary.',
      'src/lib/features/onboarding/components/onboarding-panel.svelte':
        'Onboarding panel still owns cross-step progress, persistence, and completion gating until the step flow moves into a dedicated onboarding controller.',
      'src/lib/features/onboarding/components/step-repo.svelte':
        'Repository onboarding still combines repo creation, existing repo link, and validation messaging until the create-vs-link step bodies split.',
      'src/lib/features/machines/components/machines-page.svelte':
        'Machines page still owns cache refresh, editor selection, and health probe coordination until machine detail state moves behind a page controller.',
      'src/lib/features/settings/components/workflow-scheduled-job-cron-picker.svelte':
        'Cron picker still combines visual interval controls and manual expression preview until the picker inputs split from the preview block.',
      'src/lib/features/settings/components/workflow-scheduled-jobs-panel.svelte':
        'Scheduled jobs panel still owns list loading, editor sheet state, and trigger mutations until a panel controller extracts the job actions.',
      'src/lib/features/skills/components/skill-ai-sidebar.svelte':
        'Skill AI sidebar still combines refinement session control, streaming transcript state, and verified patch rendering until the session controller splits from the result view.',
      'src/lib/features/skills/components/skill-editor-page.svelte':
        'Skill editor page still owns draft save orchestration, binding sheets, and fix-and-verify launch state until the page controller extracts those workflows.',
      'src/lib/features/skills/components/skills-page.svelte':
        'Skills page still mixes browser selection, editor bootstrap, and loading/error states until the browser shell splits from editor state.',
      'src/lib/features/ticket-detail/components/ticket-drawer.svelte':
        'Ticket drawer still owns shell layout, pane selection, and runtime status affordances until drawer chrome separates from ticket detail state.',
      'src/lib/features/ticket-detail/components/ticket-comments-thread.svelte':
        'Ticket comments thread still combines grouped activity rendering with inline comment editing and composer flow until discussion actions split from the timeline body.',
      'src/lib/features/ticket-detail/components/ticket-run-history-panel.svelte':
        'Ticket run history panel still combines run rail selection, live transcript rendering, and follow-live controls until the rail and transcript panes split.',
      'src/lib/features/tickets/components/new-ticket-dialog.svelte':
        'Ticket creation dialog still owns async option loading and the multi-picker metadata row until the metadata controls split from the form body.',
      'src/lib/features/tickets/components/tickets-page.svelte':
        'Tickets page still owns board bootstrap, stream orchestration, and drawer coordination until board data flow extracts into a page controller.',
      'src/lib/features/workflows/components/workflows-page.svelte':
        'Workflows page still coordinates list selection, editor state, history loading, and skill bindings until workflow page actions move behind a controller.',
    },
  },
  {
    name: 'Feature modules',
    match: (filePath) => /^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/.test(filePath),
    softLimit: 200,
    hardLimit: 300,
    allowlist: {
      'src/lib/features/board/components/board-page-controls.test.ts':
        'Board controls coverage still bundles filter, selection, sort, and bulk-action matrices until scenario builders split those behavior families.',
      'src/lib/features/board/components/board-page-runtime-and-drawer.test.ts':
        'Board runtime-and-drawer coverage still mixes stream recovery and drawer interaction scenarios until shared board harness helpers extract.',
      'src/lib/features/chat/project-conversation-controller.svelte.ts':
        'Project conversation controller still owns tab persistence, stream lifecycle, and action proposal state until per-tab runtime state splits from restore orchestration.',
      'src/lib/features/chat/project-conversation-controller.test.ts':
        'Project conversation controller coverage still spans restore, queueing, and runtime recovery scenarios until controller test builders split those flows.',
      'src/lib/features/dashboard/components/org-dashboard.test.ts':
        'Dashboard regression coverage still combines initial load, refresh invalidation, and advisor rendering cases until dashboard test fixtures split by panel.',
      'src/lib/features/skills/components/skill-bundle-editor.ts':
        'Skill bundle editor still centralizes draft mutations, tree normalization, and path rewrites until file-tree helpers split from draft reducers.',
      'src/lib/features/ticket-detail/drawer-state.svelte.test.ts':
        'Ticket drawer state coverage still mixes reference cache recovery with run transcript selection until drawer-state fixtures split by state slice.',
      'src/lib/features/ticket-detail/drawer-state.svelte.ts':
        'Ticket drawer state still owns reference caching, live refresh wiring, and transcript selection until run history state splits from drawer references.',
      'src/lib/features/ticket-detail/run-transcript.test.ts':
        'Run transcript coverage still bundles lifecycle, step, and trace reducer scenarios until transcript fixtures split by event family.',
      'src/lib/features/ticket-detail/run-transcript.ts':
        'Run transcript reducer still folds lifecycle, step, and trace events in one module until event-family reducers extract behind a shared transcript assembler.',
      'src/lib/features/workflows/components/harness-ai-sidebar-streaming.test.ts':
        'Harness AI streaming coverage still keeps long-stream SSE assertions in one integration test until sidebar stream helpers extract.',
      'src/lib/features/workflows/components/workflows-page.test.ts':
        'Workflow page coverage still mixes editor, history, and binding interactions until page-level workflow test helpers split by surface.',
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
