process.env.TZ = 'UTC';

const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');

module.exports = {
  ...require('./.config/jest.config'),
  testEnvironmentOptions: {
    url: 'http://localhost:3000',
  },
  transformIgnorePatterns: [nodeModulesToTransform([...grafanaESModules])],
};
