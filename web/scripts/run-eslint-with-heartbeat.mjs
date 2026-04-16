import { spawn } from 'node:child_process'

const heartbeatMs = Math.max(1_000, Number(process.env.ESLINT_HEARTBEAT_MS ?? 15_000))
const eslintArgs = process.argv.slice(2)

const child = spawn('pnpm', ['exec', 'eslint', ...eslintArgs], {
  stdio: 'inherit',
  env: process.env,
})

const heartbeat = setInterval(() => {
  console.log(`[eslint-heartbeat] ${new Date().toISOString()} eslint still running`)
}, heartbeatMs)

child.on('exit', (code, signal) => {
  clearInterval(heartbeat)
  if (signal) {
    process.kill(process.pid, signal)
    return
  }
  process.exit(code ?? 1)
})

child.on('error', (error) => {
  clearInterval(heartbeat)
  console.error('[eslint-heartbeat] failed to start eslint:', error)
  process.exit(1)
})
