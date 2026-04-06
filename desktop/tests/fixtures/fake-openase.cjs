#!/usr/bin/env node
const http = require('node:http')

const args = process.argv.slice(2)
const host = readArgument('--host') ?? '127.0.0.1'
const port = Number(readArgument('--port') ?? 0)
const configPath = readArgument('--config') ?? ''
const readyDelayMs = Number(process.env.FAKE_OPENASE_READY_DELAY_MS ?? 0)
const exitAfterReadyMs = Number(process.env.FAKE_OPENASE_EXIT_AFTER_READY_MS ?? 0)
const failImmediately = process.env.FAKE_OPENASE_FAIL_IMMEDIATELY === '1'
const readyAt = Date.now() + readyDelayMs

if (!port) {
  console.error('fake-openase requires --port')
  process.exit(1)
}

if (failImmediately) {
  console.error('fake-openase configured to fail immediately')
  process.exit(1)
}

const server = http.createServer((request, response) => {
  if (request.url === '/healthz' || request.url === '/api/v1/healthz') {
    if (Date.now() < readyAt) {
      response.writeHead(503, { 'content-type': 'application/json' })
      response.end(JSON.stringify({ status: 'starting' }))
      return
    }

    response.writeHead(200, { 'content-type': 'application/json' })
    response.end(JSON.stringify({ status: 'ok', config_path: configPath }))
    return
  }

  if (request.url === '/api/v1/fixture') {
    response.writeHead(200, { 'content-type': 'application/json' })
    response.end(JSON.stringify({ message: 'fixture-ok' }))
    return
  }

  response.writeHead(200, { 'content-type': 'text/html; charset=utf-8' })
  response.end(`<!doctype html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <title>OpenASE Desktop Fixture</title>
      <style>
        body { margin: 0; font-family: Georgia, serif; background: #0a1216; color: #edf8f5; }
        main { min-height: 100vh; display: grid; place-items: center; }
        section { width: min(720px, 100%); padding: 40px; }
        h1 { font-size: 3rem; margin: 0 0 16px; }
        p { color: #9bb5af; line-height: 1.6; }
      </style>
    </head>
    <body>
      <main>
        <section>
          <h1>OpenASE workspace is running.</h1>
          <p>Config: ${configPath || 'none'}</p>
          <p id="api-status">Checking API...</p>
        </section>
      </main>
      <script>
        fetch('/api/v1/fixture')
          .then((response) => response.json())
          .then((payload) => {
            document.getElementById('api-status').textContent = 'API connected: ' + payload.message
          })
          .catch((error) => {
            document.getElementById('api-status').textContent = 'API failed: ' + error.message
          })
      </script>
    </body>
  </html>`)
})

server.listen(port, host, () => {
  console.log(`fake-openase listening on http://${host}:${port}`)
  if (exitAfterReadyMs > 0) {
    const delay = Math.max(readyDelayMs + exitAfterReadyMs, exitAfterReadyMs)
    setTimeout(() => {
      process.exit(0)
    }, delay)
  }
})

process.on('SIGTERM', shutdown)
process.on('SIGINT', shutdown)

function shutdown() {
  server.close(() => process.exit(0))
}

function readArgument(flag) {
  const index = args.indexOf(flag)
  if (index === -1) {
    return null
  }
  return args[index + 1] ?? null
}
