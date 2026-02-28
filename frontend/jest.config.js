// jest.config.js — compatible with Jest 25 (which does not support .ts config files
// or async config functions). This file replaces the broken jest.config.ts setup.

const path = require('path')

/** @type {import('jest').Config} */
const config = {
  coverageProvider: 'v8',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  testMatch: [
    '**/__tests__/**/*.[jt]s?(x)',
    '**/?(*.)+(spec|test).[jt]s?(x)',
  ],
  testPathIgnorePatterns: ['/node_modules/', '/.next/', '/e2e/'],
  moduleNameMapper: {
    // CSS / images — re-use next/jest stubs
    '^.+\\.module\\.(css|sass|scss)$': path.resolve(
      __dirname,
      'node_modules/next/dist/build/jest/object-proxy.js',
    ),
    '^.+\\.(css|sass|scss)$': path.resolve(
      __dirname,
      'node_modules/next/dist/build/jest/__mocks__/styleMock.js',
    ),
    '^.+\\.(png|jpg|jpeg|gif|webp|avif|ico|bmp|svg)$': path.resolve(
      __dirname,
      'node_modules/next/dist/build/jest/__mocks__/fileMock.js',
    ),
    'next/font/(.*)': path.resolve(
      __dirname,
      'node_modules/next/dist/build/jest/__mocks__/nextFontMock.js',
    ),
    '^server-only$': path.resolve(
      __dirname,
      'node_modules/next/dist/build/jest/__mocks__/empty.js',
    ),
    // Path alias
    '^@/(.*)$': '<rootDir>/$1',
  },
  transform: {
    '^.+\\.[jt]sx?$': [
      path.resolve(
        __dirname,
        'node_modules/next/dist/build/swc/jest-transformer',
      ),
      {
        // SWC config: treat as TypeScript + React
      },
    ],
  },
  transformIgnorePatterns: [
    '/node_modules/',
    '^.+\\.module\\.(css|sass|scss)$',
  ],
}

module.exports = config
