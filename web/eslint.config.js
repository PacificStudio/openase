import js from '@eslint/js'
import { defineConfig } from 'eslint/config'
import importPlugin from 'eslint-plugin-import'
import sonarjs from 'eslint-plugin-sonarjs'
import svelte from 'eslint-plugin-svelte'
import globals from 'globals'
import ts from 'typescript-eslint'
import svelteConfig from './svelte.config.js'

export default defineConfig(
  {
    ignores: ['.svelte-kit/**', 'build/**', 'dist/**', 'node_modules/**'],
  },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  ...svelte.configs.prettier,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    settings: {
      'import/resolver': {
        node: {
          extensions: ['.js', '.mjs', '.cjs', '.ts', '.svelte'],
        },
        typescript: {
          alwaysTryTypes: true,
          project: './tsconfig.json',
        },
      },
    },
  },
  {
    files: ['**/*.ts', '**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    plugins: {
      import: importPlugin,
      sonarjs,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      complexity: ['warn', 10],
      'import/no-cycle': ['error', { ignoreExternal: true }],
      'max-lines-per-function': ['warn', { max: 60, skipBlankLines: true, skipComments: true }],
      'no-undef': 'off',
      'sonarjs/cognitive-complexity': ['warn', 15],
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
        svelteConfig,
      },
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    rules: {
      'svelte/no-at-html-tags': 'error',
      'svelte/no-unused-svelte-ignore': 'error',
      'svelte/no-navigation-without-resolve': 'off',
      'svelte/no-useless-mustaches': 'off',
      'svelte/prefer-svelte-reactivity': 'off',
      'svelte/require-each-key': 'off',
    },
  },
  {
    files: ['src/routes/**/+page.svelte'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: ['src/routes/**/+layout.svelte'],
    rules: {
      'max-lines': ['error', { max: 300, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: ['src/lib/features/**/*.svelte', 'src/lib/features/**/*.{js,ts,mjs,cjs}'],
    rules: {
      'max-lines': ['error', { max: 300, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: ['src/lib/components/layout/**/*.svelte'],
    rules: {
      'max-lines': ['error', { max: 300, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: ['src/lib/components/ui/**/*.svelte'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: [
      'src/lib/features/agents/components/agent-drawer.svelte',
      'src/lib/features/board/components/board-page-controls.test.ts',
      'src/lib/features/dashboard/components/org-dashboard.svelte',
      'src/lib/features/dashboard/components/org-dashboard.test.ts',
      'src/lib/features/onboarding/components/onboarding-panel.svelte',
      'src/lib/features/onboarding/components/step-repo.svelte',
      'src/lib/features/machines/components/machines-page.svelte',
      'src/lib/features/project-updates/components/project-update-thread-card.svelte',
      'src/lib/features/skills/components/skill-bundle-editor.ts',
      'src/lib/features/skills/components/skill-ai-sidebar.test.ts',
      'src/lib/features/skills/components/skill-editor-page.svelte',
      'src/lib/features/skills/components/skill-refinement-transcript.ts',
      'src/lib/features/ticket-detail/components/ticket-run-history-panel.svelte',
      'src/lib/features/tickets/components/new-ticket-dialog.svelte',
      'src/lib/features/tickets/components/tickets-page.svelte',
      'src/lib/features/ticket-detail/components/ticket-drawer.svelte',
      'src/lib/features/ticket-detail/drawer-state.svelte.test.ts',
      'src/lib/features/ticket-detail/drawer-state.svelte.ts',
      'src/lib/features/ticket-detail/run-transcript.test.ts',
      'src/lib/features/ticket-detail/run-transcript.ts',
      'src/lib/features/workflows/components/harness-ai-sidebar-streaming.test.ts',
      'src/lib/features/workflows/components/workflows-page.test.ts',
      'src/lib/features/chat/project-conversation-controller.svelte.ts',
      'src/lib/features/chat/project-conversation-controller.test.ts',
      'src/lib/features/workflows/components/workflows-page.svelte',
    ],
    rules: {
      'max-lines': 'off',
    },
  },
  {
    files: ['**/*.{js,cjs,mjs,ts}'],
    rules: {
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'prefer-const': 'error',
    },
  },
)
