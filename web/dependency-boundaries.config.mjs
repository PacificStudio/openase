export const aliasRoots = {
  $lib: 'src/lib',
  $components: 'src/lib/components',
  $ui: 'src/lib/components/ui',
  $utils: 'src/lib/utils',
  $hooks: 'src/lib/hooks',
}

export const boundaryRules = [
  {
    name: 'ui-primitives-stay-foundational',
    from: /^src\/lib\/components\/ui\//,
    disallow: [
      {
        to: /^src\/lib\/components\/layout\//,
        reason: 'UI primitives cannot depend on app-shell layout components.',
      },
      {
        to: /^src\/lib\/features\//,
        reason: 'UI primitives cannot depend on feature-layer code.',
      },
      {
        to: /^src\/routes\//,
        reason: 'UI primitives cannot depend on route-layer code.',
      },
    ],
  },
  {
    name: 'layout-does-not-know-features-or-routes',
    from: /^src\/lib\/components\/layout\//,
    disallow: [
      {
        to: /^src\/lib\/features\//,
        reason: 'Layout components cannot depend on feature implementations.',
      },
      {
        to: /^src\/routes\//,
        reason: 'Layout components cannot depend on route-layer code.',
      },
    ],
  },
  {
    name: 'features-do-not-import-routes',
    from: /^src\/lib\/features\/([^/]+)\//,
    disallow: [
      {
        to: /^src\/routes\//,
        reason: 'Feature modules cannot depend on route files.',
      },
    ],
  },
  {
    name: 'features-only-cross-import-public-entrypoints',
    from: /^src\/lib\/features\/([^/]+)\//,
    check({ fromMatch, toPath }) {
      const targetMatch = /^src\/lib\/features\/([^/]+)\/(.+)$/.exec(toPath)
      if (!targetMatch) {
        return null
      }

      const [, sourceFeature] = fromMatch
      const [, targetFeature, targetRemainder] = targetMatch

      if (sourceFeature === targetFeature) {
        return null
      }

      if (targetRemainder === 'index.ts' || targetRemainder === 'public.ts') {
        return null
      }

      return 'Feature modules may only import another feature through its public entrypoint.'
    },
    allowlist: {
      'src/lib/features/board/components/board-page.test.ts':
        'Legacy board coverage still reuses the tickets page until board/tickets view extraction lands.',
      'src/lib/features/tickets/components/tickets-page.svelte':
        'Legacy tickets page still composes board internals until the board feature boundary is extracted.',
    },
  },
  {
    name: 'routes-do-not-import-route-implementations',
    from: /^src\/routes\//,
    disallow: [
      {
        to: /^src\/routes\/.*\+(page|layout)\.(svelte|ts|js)$/,
        reason: 'Route files are assembly-only and must not depend on other route implementations.',
      },
    ],
  },
  {
    name: 'pages-do-not-reach-into-api-boundary',
    from: /^src\/routes\/.+\/\+(page|layout)\.svelte$|^src\/routes\/\+(page|layout)\.svelte$/,
    disallow: [
      {
        to: /^src\/lib\/api\//,
        reason: 'Route files must compose features instead of owning API or SSE integrations.',
      },
    ],
    allowlist: {
      'src/routes/+page.svelte':
        'Legacy dashboard route still owns API/SSE wiring until the dashboard refactor lands.',
      'src/routes/(app)/orgs/+page.svelte':
        'Legacy workspace organizations route still owns API-backed dashboard wiring until org dashboard extraction lands.',
      'src/routes/(app)/orgs/[orgId]/+page.svelte':
        'Legacy organization overview route still owns API-backed dashboard wiring until org dashboard extraction lands.',
      'src/routes/ticket/+page.svelte':
        'Legacy ticket detail route still owns API/SSE wiring until the ticket refactor lands.',
    },
  },
]
