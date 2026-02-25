import nextJest from 'next/jest.js'

const createJestConfig = nextJest({ dir: './' })

const config = {
  coverageProvider: 'v8',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  testMatch: [
    '**/__tests__/**/*.[jt]s?(x)',
    '**/?(*.)+(spec|test).[jt]s?(x)',
  ],
  testPathIgnorePatterns: [
    '/node_modules/',
    '/.next/',
    '/e2e/',
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/$1',
  },
}

export default createJestConfig(config)
