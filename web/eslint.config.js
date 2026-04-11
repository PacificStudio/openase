import js from '@eslint/js'
import { defineConfig } from 'eslint/config'
import importPlugin from 'eslint-plugin-import'
import sonarjs from 'eslint-plugin-sonarjs'
import svelte from 'eslint-plugin-svelte'
import globals from 'globals'
import ts from 'typescript-eslint'
import { fileBudgetLimits } from './file-budgets.config.mjs'
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
      'max-lines': [
        'error',
        { max: fileBudgetLimits.routePage.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/routes/**/+layout.svelte'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.routeLayout.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/features/**/*.svelte'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.featureComponent.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/features/**/*.test.{js,ts,mjs,cjs}'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.featureTest.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/features/**/*.svelte.{ts,js}'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.featureStateModule.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/features/**/*.{js,ts,mjs,cjs}'],
    ignores: ['src/lib/features/**/*.test.{js,ts,mjs,cjs}', 'src/lib/features/**/*.svelte.{ts,js}'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.featureModule.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/components/layout/**/*.svelte'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.layoutComponent.hard, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  {
    files: ['src/lib/components/ui/**/*.svelte'],
    rules: {
      'max-lines': [
        'error',
        { max: fileBudgetLimits.uiPrimitive.hard, skipBlankLines: true, skipComments: true },
      ],
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
