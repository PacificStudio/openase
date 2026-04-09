import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import { svelteTesting } from '@testing-library/svelte/vite'
import { sveltekit } from '@sveltejs/kit/vite'
import { loadEnv } from 'vite'
import { configDefaults, defineConfig } from 'vitest/config'
import { handleMockApi } from './src/lib/testing/e2e/mock-api'

const defaultDevHost = '127.0.0.1'
const defaultDevPort = 4173
const remendCompatPath = fileURLToPath(
  new URL('./src/lib/features/markdown/remend-compat.ts', import.meta.url),
)

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.OPENASE_DEV_PROXY_TARGET?.trim()
  const devHost = env.OPENASE_DEV_HOST?.trim() || defaultDevHost
  const devPort = Number(env.OPENASE_DEV_PORT || defaultDevPort)
  const e2eMockEnabled = env.OPENASE_E2E_MOCK === '1'

  return {
    plugins: [tailwindcss(), sveltekit(), svelteTesting(), e2eMockApiPlugin(e2eMockEnabled)],
    resolve: {
      alias: [
        // `streamdown-svelte` currently expects a parser class that `remend@1.3.0`
        // does not export, so route only its `remend` import through a local shim.
        { find: /^remend$/, replacement: remendCompatPath },
      ],
    },
    server: proxyTarget
      ? {
          host: devHost,
          port: devPort,
          proxy: {
            '/api': {
              target: proxyTarget,
              // Preserve the browser-facing dev origin so backend redirects and
              // auto-derived URLs stay on the Vite host instead of flipping to
              // the raw proxy target.
              changeOrigin: false,
              xfwd: true,
            },
          },
        }
      : {
          host: devHost,
          port: devPort,
        },
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      exclude: [...configDefaults.exclude, 'tests/e2e/**'],
      server: {
        deps: {
          inline: ['streamdown-svelte', 'katex'],
        },
      },
    },
  }
})

function e2eMockApiPlugin(enabled: boolean) {
  return {
    name: 'openase-e2e-mock-api',
    apply: 'serve' as const,
    configureServer(server: { middlewares: { use: (handler: NodeMiddleware) => void } }) {
      if (!enabled) {
        return
      }

      server.middlewares.use(async (req, res, next) => {
        const rawUrl = req.url ?? ''
        if (!rawUrl.startsWith('/api/v1/')) {
          next()
          return
        }

        const url = new URL(rawUrl, `http://${req.headers.host ?? '127.0.0.1'}`)
        const request = await toWebRequest(req, url)
        const mockedResponse = await handleMockApi(request, url)
        if (!mockedResponse) {
          next()
          return
        }

        await writeWebResponse(res, mockedResponse)
      })
    },
  }
}

type NodeMiddleware = (
  req: import('node:http').IncomingMessage,
  res: import('node:http').ServerResponse,
  next: () => void,
) => void | Promise<void>

async function toWebRequest(req: import('node:http').IncomingMessage, url: URL): Promise<Request> {
  const method = req.method ?? 'GET'
  const headers = new Headers()
  for (const [key, value] of Object.entries(req.headers)) {
    if (Array.isArray(value)) {
      for (const item of value) {
        headers.append(key, item)
      }
      continue
    }
    if (value != null) {
      headers.set(key, value)
    }
  }

  const body =
    method === 'GET' || method === 'HEAD' ? undefined : await readIncomingMessageBody(req)

  return new Request(url, {
    method,
    headers,
    body: body ? new Blob([new Uint8Array(body)]) : undefined,
  })
}

async function readIncomingMessageBody(req: import('node:http').IncomingMessage): Promise<Buffer> {
  const chunks: Uint8Array[] = []
  for await (const chunk of req) {
    if (typeof chunk === 'string') {
      chunks.push(Buffer.from(chunk))
      continue
    }
    chunks.push(chunk)
  }
  return Buffer.concat(chunks)
}

async function writeWebResponse(
  res: import('node:http').ServerResponse,
  response: Response,
): Promise<void> {
  res.statusCode = response.status
  response.headers.forEach((value, key) => {
    res.setHeader(key, value)
  })

  if (!response.body) {
    res.end()
    return
  }

  const reader = response.body.getReader()
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      res.write(Buffer.from(value))
    }
    res.end()
  } catch (error) {
    res.destroy(error instanceof Error ? error : undefined)
    throw error
  }
}
