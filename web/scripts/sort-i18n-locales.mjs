import { readFile, readdir, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

const localesDir = fileURLToPath(new URL('../src/lib/i18n/locales/', import.meta.url))
const checkOnly = process.argv.includes('--check')

const files = await readdir(localesDir)
const jsonFiles = files.filter((entry) => entry.endsWith('.json'))

let updated = 0
for (const fileName of jsonFiles) {
  const filePath = `${localesDir}/${fileName}`
  const raw = await readFile(filePath, 'utf-8')
  const parsed = JSON.parse(raw)
  const sortedEntries = Object.entries(parsed).sort((left, right) =>
    left[0].localeCompare(right[0], 'en', { sensitivity: 'accent' }),
  )
  const reformatted = `${JSON.stringify(Object.fromEntries(sortedEntries), null, 2)}\n`
  if (raw !== reformatted) {
    updated += 1
    if (checkOnly) {
      console.error(`locale file is not sorted: ${fileName}`)
      continue
    }
    await writeFile(filePath, reformatted, 'utf-8')
    console.log(`sorted ${fileName}`)
  }
}

if (updated === 0) {
  console.log('locales already sorted')
} else if (checkOnly) {
  process.exit(1)
}
