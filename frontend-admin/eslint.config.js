import js from '@eslint/js';
import tsParser from '@typescript-eslint/parser';
import tsPlugin from '@typescript-eslint/eslint-plugin';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import prettier from 'eslint-plugin-prettier/recommended';
import globals from 'globals';

// Flat-config replacement for the legacy .eslintrc.js the project carried from
// the Falcon template. ESLint 9 dropped the eslintrc format; the CI `Lint` job
// fails to even load a config without this file. Rule set is intentionally a
// straight port of the prior config — no new style enforcement.
export default [
  {
    ignores: [
      'build/**',
      'dist/**',
      'coverage/**',
      'node_modules/**',
      'public/**',
      'src/reference/**'
    ]
  },
  js.configs.recommended,
  {
    ...react.configs.flat.recommended,
    settings: { react: { version: 'detect' } }
  },
  prettier,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: { ecmaFeatures: { jsx: true } }
    },
    plugins: { '@typescript-eslint': tsPlugin },
    rules: {
      ...tsPlugin.configs.recommended.rules,
      // TS itself owns symbol resolution; the base rule fires on every DOM
      // lib type and React namespace reference.
      'no-undef': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-unused-expressions': 'off',
      '@typescript-eslint/no-unused-vars': 'off'
    }
  },
  {
    plugins: { 'react-hooks': reactHooks },
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      parserOptions: { ecmaFeatures: { jsx: true } },
      globals: {
        ...globals.browser,
        ...globals.node,
        process: true
      }
    },
    rules: {
      'react/no-unescaped-entities': 'off',
      'react/prop-types': 'off',
      'react/display-name': 'off',
      'react/react-in-jsx-scope': 'off',
      'react-hooks/exhaustive-deps': 'off',
      'react-hooks/rules-of-hooks': 'error',
      'no-unused-vars': 'off',
      'no-useless-catch': 'off',
      'prettier/prettier': ['error', { endOfLine: 'auto' }]
    }
  }
];
