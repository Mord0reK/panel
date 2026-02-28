// Jest 25 does not support async config functions returned by next/jest's
// createJestConfig. We configure Jest manually here with the same settings
// that next/jest would have injected.

const config = {
  coverageProvider: 'v8' as const,
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
    // CSS / images
    '^.+\\.module\\.(css|sass|scss)$': require.resolve(
      'next/dist/build/jest/object-proxy.js',
    ),
    '^.+\\.(css|sass|scss)$': require.resolve(
      'next/dist/build/jest/__mocks__/styleMock.js',
    ),
    '^.+\\.(png|jpg|jpeg|gif|webp|avif|ico|bmp|svg)$': require.resolve(
      'next/dist/build/jest/__mocks__/fileMock.js',
    ),
    // next/font
    'next/font/(.*)': require.resolve(
      'next/dist/build/jest/__mocks__/nextFontMock.js',
    ),
    // Alias
    '^@/(.*)$': '<rootDir>/$1',
  },
  transform: {
    '^.+\\.[jt]sx?$': [
      require.resolve('next/dist/build/swc/jest-transformer'),
      {
        // minimal SWC config
      },
    ],
  },
  transformIgnorePatterns: [
    '/node_modules/',
    '^.+\\.module\\.(css|sass|scss)$',
  ],
}

module.exports = config
