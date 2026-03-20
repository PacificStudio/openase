import adapter from '@sveltejs/adapter-static'

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter({
      assets: '../internal/webui/static',
      pages: '../internal/webui/static',
      fallback: 'index.html',
    }),
    alias: {
      $components: 'src/lib/components',
      $hooks: 'src/lib/hooks',
      $ui: 'src/lib/components/ui',
      $utils: 'src/lib/utils',
    },
  },
  vitePlugin: {
    dynamicCompileOptions: ({ filename }) =>
      filename.includes('node_modules') ? undefined : { runes: true },
  },
}

export default config
