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
    ignores: ['src/routes/+page.svelte', 'src/routes/ticket/+page.svelte'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }],
    },
  },
  {
    files: ['src/routes/+page.svelte', 'src/routes/ticket/+page.svelte'],
    rules: {
      'max-lines': ['warn', { max: 250, skipBlankLines: true, skipComments: true }],
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
      'src/lib/features/board/components/board-page-controls.test.ts',
      'src/lib/features/skills/components/skill-ai-sidebar.test.ts',
      'src/lib/features/skills/components/skill-editor-page.test.ts',
      'src/lib/features/workflows/components/harness-ai-sidebar-streaming.test.ts',
      'src/lib/features/workflows/components/workflows-page.test.ts',
      'src/lib/features/chat/project-conversation-controller.test.ts',
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
