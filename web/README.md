# OpenASE Frontend Conventions

The `web` package is the SvelteKit frontend embedded into the OpenASE Go binary. This README is the working contract for how frontend code is organized, split, and blocked in CI.

## Layering

OpenASE frontend code must follow a one-way dependency stack. The first line is the coarse-grained layer boundary; the second line expands what usually sits inside a feature implementation:

```text
ui -> layout -> features -> routes
types/mappers -> api/stores -> components -> routes
```

- `src/lib/components/ui/`: primitive UI only. No feature semantics, no API calls, no route knowledge.
- `src/lib/components/layout/`: app shell building blocks such as sidebars, headers, drawers, and empty states.
- `src/lib/features/<feature>/`: feature-owned API adapters, stores, mappers, types, and components.
- `src/routes/**/+page.svelte`: assembly layer only. Pages wire route params, feature stores, and page sections together.

## Route Responsibilities

`routes/**/+page.svelte` and `routes/**/+layout.svelte` are not implementation buckets.

- Allowed: page composition, route params, load output wiring, small view-only helpers.
- Move out immediately: API fetch wrappers, SSE protocol handling, feature-specific state machines, large type blocks, and deep rendering branches.
- If a route starts reading like a feature implementation, split a feature folder first and keep the route as the table of contents.

## Feature-First Structure

Target structure for new work:

```text
src/
├── routes/
│   ├── (app)/
│   └── (setup)/
└── lib/
    ├── components/
    │   ├── ui/
    │   └── layout/
    ├── features/
    │   ├── board/
    │   ├── dashboard/
    │   ├── ticket-detail/
    │   ├── agents/
    │   └── workflows/
    ├── api/
    ├── stores/
    └── utils/
```

Feature modules own their `api.ts`, `stores.ts`, `types.ts`, `mappers.ts`, and `components/`. Cross-feature imports must go through a public entrypoint such as `index.ts`.

## File Budgets

These budgets are enforced by `npm run lint:structure` and mirrored in ESLint where practical.

| File type                           | Soft limit | Hard limit           |
| ----------------------------------- | ---------- | -------------------- |
| `routes/**/+page.svelte`            | 150        | 250                  |
| `routes/**/+layout.svelte`          | 180        | 300                  |
| `lib/features/**/*.svelte`          | 200        | 300                  |
| `lib/features/**/*.{ts,js}`         | 200        | 300                  |
| `lib/components/layout/**/*.svelte` | 200        | 300                  |
| `lib/components/ui/**/*.svelte`     | 150        | 250                  |
| single function                     | 40 target  | 60 warning threshold |

Current legacy waivers:

- `src/routes/+page.svelte`
- `src/routes/ticket/+page.svelte`

Those files are explicitly tracked as refactor debt. New oversized route files are blocked.

## Quality Gates

```sh
npm install
npm run lint
npm run lint:structure
npm run lint:deps
npm run check
npm run build
npm run ci
```

- `npm run lint`: ESLint with complexity, file-size, and cycle checks.
- `npm run lint:structure`: custom file budget enforcement with explicit waivers for current debt only.
- `npm run lint:deps`: dependency boundary enforcement for `ui -> layout -> features -> routes`.
- `npm run check`: `svelte-check` type validation.
- `npm run ci`: unified local and CI entrypoint for the frontend gate.

## Review Checklist

Every frontend PR should answer these before merge:

- Is the route file still a composition layer instead of the implementation layer?
- Did new business UI land under `lib/features/` instead of `lib/components/`?
- Are API calls, SSE merging, and feature state outside the route file?
- Did any new file cross its budget or import upward across the layer boundary?
