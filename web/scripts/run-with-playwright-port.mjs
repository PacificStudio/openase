import { spawn } from 'node:child_process'
import net from 'node:net'
import process from 'node:process'

const command = process.argv.slice(2).join(' ').trim()
if (!command) {
  console.error('Usage: node scripts/run-with-playwright-port.mjs <command...>')
  process.exit(1)
}

const host = process.env.PLAYWRIGHT_WEB_HOST ?? process.env.PLAYWRIGHT_HOST ?? '127.0.0.1'
const requestedPort = process.env.PLAYWRIGHT_WEB_PORT ?? process.env.PLAYWRIGHT_PORT
const port = requestedPort ? parsePort(requestedPort) : await findFreePort(host)
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://${host}:${port}`

const child = spawn(command, {
  stdio: 'inherit',
  shell: true,
  env: {
    ...process.env,
    PLAYWRIGHT_HOST: process.env.PLAYWRIGHT_HOST ?? host,
    PLAYWRIGHT_WEB_HOST: process.env.PLAYWRIGHT_WEB_HOST ?? host,
    PLAYWRIGHT_PORT: process.env.PLAYWRIGHT_PORT ?? String(port),
    PLAYWRIGHT_WEB_PORT: process.env.PLAYWRIGHT_WEB_PORT ?? String(port),
    PLAYWRIGHT_BASE_URL: baseURL,
  },
})

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal)
    return
  }
  process.exit(code ?? 1)
})

child.on('error', (error) => {
  console.error(error)
  process.exit(1)
})

function parsePort(raw) {
  const parsed = Number(raw)
  if (!Number.isInteger(parsed) || parsed < 1 || parsed > 65_535) {
    throw new Error(`PLAYWRIGHT_PORT must be a valid TCP port, got ${raw}`)
  }
  return parsed
}

function findFreePort(hostname) {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.unref()
    server.on('error', reject)
    server.listen(0, hostname, () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        server.close(() => reject(new Error('Failed to resolve a free Playwright port')))
        return
      }

      const { port } = address
      server.close((error) => {
        if (error) {
          reject(error)
          return
        }
        resolve(port)
      })
    })
  })
}
