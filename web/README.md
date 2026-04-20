# OpenASE Frontend Conventions

The `web` package is the SvelteKit frontend embedded into the OpenASE Go binary. This README is the working contract for how frontend code is organized, split, and blocked in CI.

## Local Dev Mode

You can run the frontend as a standalone Vite dev server and proxy API traffic to an already-running OpenASE backend on the same machine.

Default dev settings:

- frontend host: `127.0.0.1`
- frontend port: `4173`
- backend proxy target: unset by default

When `OPENASE_DEV_PROXY_TARGET` is set, Vite proxies `/api/*` to that backend. This covers normal JSON APIs and SSE streams because the frontend already uses relative `/api/...` paths.

Example against a repo-local OpenASE backend on `127.0.0.1:19836`:

```sh
cd web
PATH=$HOME/.nvm/versions/node/v22.22.1/bin:$PATH \
OPENASE_DEV_PROXY_TARGET=http://127.0.0.1:19836 \
pnpm dev
```

Then open:

```text
http://127.0.0.1:4173
```

Optional overrides:

- `OPENASE_DEV_HOST`
- `OPENASE_DEV_PORT`
- `OPENASE_DEV_PROXY_TARGET`

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

These budgets are enforced by `pnpm run lint:structure` and mirrored in ESLint where practical.

| File type                           | Soft limit | Hard limit           |
| ----------------------------------- | ---------- | -------------------- |
| `routes/**/+page.svelte`            | 150        | 250                  |
| `routes/**/+layout.svelte`          | 180        | 300                  |
| `lib/features/**/*.test.{ts,js}`    | 300        | 650                  |
| `lib/features/**/*.svelte.{ts,js}`  | 250        | 400                  |
| `lib/features/**/*.svelte`          | 200        | 350                  |
| `lib/features/**/*.{ts,js}`         | 200        | 325                  |
| `lib/testing/**/*.{ts,js}`          | 350        | 650                  |
| `lib/components/layout/**/*.svelte` | 200        | 300                  |
| `lib/components/ui/**/*.svelte`     | 150        | 250                  |
| single function                     | 40 target  | 60 warning threshold |

There are no per-file budget waivers. Budgets live in one shared category definition that drives both `lint:structure` and the mirrored ESLint `max-lines` rules, so any recurring exception should be promoted into a named category instead of growing an allowlist.

## Quality Gates

```sh
pnpm install
pnpm run lint
pnpm run lint:mobile
pnpm run lint:structure
pnpm run lint:deps
pnpm run check
pnpm run build
pnpm run ci
```

- `pnpm run lint`: ESLint with complexity, file-size, and cycle checks.
- `pnpm run lint:i18n`: fails on newly introduced hardcoded user-visible strings that do not go through the shared i18n layer.
- `pnpm run lint:mobile`: validates that every project route declares a mobile support policy and that responsive routes wire into the mobile regression templates.
- `pnpm run lint:structure`: custom file budget enforcement with first-class categories for routes, feature tests, testing support modules, state modules, and UI layers.
- `pnpm run lint:deps`: dependency boundary enforcement for `ui -> layout -> features -> routes` with no waiver path.
- `pnpm run check`: `svelte-check` type validation.
- `pnpm run ci`: unified local and CI entrypoint for the frontend gate.

## i18n Rules

OpenASE now ships a lightweight frontend i18n layer under `src/lib/i18n/` with `en` and `zh` locale support.

- Runtime access: use `i18nStore.t('some.key')` in Svelte and `translate(locale, 'some.key')` in pure TypeScript helpers.
- Runtime switching: the current language is selectable from the top-right user menu and persisted in `localStorage`.
- Page titles: use `pageTitle(...)` so the localized title stays consistent with the app suffix.

Strings that must go through i18n:

- visible button, menu, link, badge, dialog, and empty-state copy
- page titles, section headings, helper copy, and status text shown to users
- user-facing accessibility text such as `aria-label`, `title`, `placeholder`, and translated `alt`
- labels or descriptions declared in TypeScript for navigation, menus, and other UI metadata

Strings that may stay literal when they are technical data rather than product copy:

- URLs, routes, API paths, IDs, protocol constants, and status codes
- CLI commands, shell snippets, file paths, and code samples that users must copy exactly
- test fixtures, mocks, generated files, and non-UI support code

When a literal exemption is truly necessary in scanned UI code:

- prefer the shared allowlist patterns in `i18n-check.config.json`
- otherwise add `i18n-exempt` on the same line or the line immediately above the literal and keep the exemption narrowly scoped
- do not use exemptions for normal product copy just to bypass translation work

## i18n Scanner

`pnpm run lint:i18n` runs `scripts/check-i18n.mjs`.

- Default mode scans the full frontend source tree and suppresses only the reviewed legacy backlog recorded in `i18n-check.baseline.json`.
- `node scripts/check-i18n.mjs --diff --base-ref origin/main` limits the scan to the current branch diff when you want a focused local pass.
- `node scripts/check-i18n.mjs --write-baseline` refreshes `i18n-check.baseline.json`; only do this after reviewing the current backlog and intentionally accepting the remaining untranslated surfaces.
- The scanner is section-aware for Svelte files: markup text/attributes are checked in template regions, while suspicious `label` / `title` / `description` style assignments are checked in script regions.
- Violations print file, line, reason, and the offending literal so CI fails with actionable output.

Baseline policy:

- treat `i18n-check.baseline.json` as a shrinking migration ledger, not a dump for fresh copy
- when you translate an existing legacy surface, make sure its offense entries disappear from the baseline on the next refresh
- do not add new user-visible literals and then "fix" CI by updating the baseline; route the string through i18n instead

When adding new copy:

1. add the key to both locale dictionaries in `src/lib/i18n/index.ts`
2. replace the literal use site with `i18nStore.t(...)` or `translate(...)`
3. run `pnpm run lint` and `pnpm run check`
4. if you intentionally cleaned up older untranslated surfaces in the same area, refresh the baseline so it keeps shrinking rather than preserving resolved debt

## Mobile Route Policy

All project routes under `src/routes/(app)/orgs/[orgId]/projects/[projectId]` must declare a support policy in `tests/e2e/mobile/policies.js`.

- `mobile-supported`: the route must pass phone and tablet mobile smoke + interaction coverage.
- `tablet-supported`: the route must pass tablet coverage but can opt out of phone layouts.
- `desktop-only`: the route is blocked from mobile coverage and must include an explicit reason.

When adding a new responsive page:

1. add the route policy entry in `tests/e2e/mobile/policies.js`
2. choose an existing `interaction.kind` template or add a new reusable one in `tests/e2e/mobile/interactions.spec.ts`
3. list the route's critical controls so the smoke/layout suite can catch overlap and reachability regressions
4. run `pnpm run test:e2e:mobile`

## Review Checklist

Every frontend PR should answer these before merge:

- Is the route file still a composition layer instead of the implementation layer?
- Did new business UI land under `lib/features/` instead of `lib/components/`?
- Are API calls, SSE merging, and feature state outside the route file?
- Did any new file cross its budget or import upward across the layer boundary?
