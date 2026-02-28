'use strict'

// Compatibility shim: adapts Jest 25 flat-config format to the format
// expected by jest-environment-jsdom@30 (which wraps @jest/environment-jsdom-abstract@30).
//
// Jest 25 runner: new TestEnvironment(flatConfig, { console, testPath, ... })
// jest-environment-jsdom-abstract@30 expects: config.projectConfig.testEnvironmentOptions
//
// This shim normalises the config before it reaches the abstract env.

const abstractEnvPath = require.resolve(
  '@jest/environment-jsdom-abstract',
)
const abstractModule = require(abstractEnvPath)
const AbstractEnv = abstractModule.default || abstractModule

const { JSDOM, ResourceLoader, VirtualConsole } = require('jsdom')

class CompatJSDOMEnvironment extends AbstractEnv {
  constructor(config, context) {
    // Jest 25 passes a flat config object (no projectConfig wrapper).
    // Jest 30 passes { projectConfig, globalConfig }.
    const normalised = config && config.projectConfig
      ? config
      : {
          ...config,
          projectConfig: {
            ...config,
            testEnvironmentOptions: (config && config.testEnvironmentOptions) || {},
          },
          globalConfig: {},
        }

    super(normalised, context || {}, { JSDOM, ResourceLoader, VirtualConsole })
  }
}

module.exports = CompatJSDOMEnvironment
