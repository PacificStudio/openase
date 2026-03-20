# OpenASE Web UI

The `web` package is the SvelteKit frontend embedded into the OpenASE Go binary at build time.

## Common Commands

```sh
pnpm install
pnpm run dev
pnpm run build
pnpm run lint
pnpm run format:check
pnpm run check
```

## Quality Gates

- `pnpm run lint` runs ESLint across Svelte, TypeScript, and config files.
- `pnpm run format` / `pnpm run format:check` apply or verify Prettier formatting.
- `pnpm run check` runs `svelte-check` against the workspace TypeScript config.
