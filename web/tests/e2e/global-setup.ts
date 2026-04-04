import { resetPerfResults } from './perf'

async function globalSetup() {
  await resetPerfResults()
}

export default globalSetup
