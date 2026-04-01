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
    allowlist: {
      'src/routes/+page.svelte':
        'Legacy dashboard route remains oversized until frontend refactor phase 1.',
      'src/routes/(app)/orgs/[orgId]/+page.svelte':
        'Legacy organization overview route remains oversized until org dashboard extraction lands.',
      'src/routes/ticket/+page.svelte':
        'Legacy ticket detail route remains oversized until ticket feature extraction lands.',
    },
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
      'src/lib/features/settings/components/workflow-scheduled-job-cron-picker.svelte':
        'Cron picker keeps presets and field editor together until scheduled job form extraction lands.',
      'src/lib/features/skills/components/skill-editor-page.svelte':
        'Skill editor page still hosts save/binding orchestration until controller extraction lands.',
      'src/lib/features/tickets/components/tickets-page.svelte':
        'Tickets page still owns board loading and stream orchestration until board controller extraction lands.',
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
      'src/lib/features/skills/components/skill-bundle-editor.ts':
        'Skill bundle draft helpers remain centralized until tree and path utilities are split out.',
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
