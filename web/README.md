# OpenASE Web UI

The `web` package is the SvelteKit frontend embedded into the OpenASE Go binary at build time.

## Common Commands

```sh
npm install
npm run dev
npm run build
npm run lint
npm run format:check
npm run check:svelte
```

## Quality Gates

- `npm run lint` runs ESLint across Svelte, TypeScript, and config files.
- `npm run format` / `npm run format:check` apply or verify Prettier formatting.
- `npm run check:svelte` runs `svelte-check` against the workspace TypeScript config.
