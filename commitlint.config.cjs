/** @type {import('@commitlint/types').Configuration} */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'header-max-length': [2, 'always', 200],
  },
};
