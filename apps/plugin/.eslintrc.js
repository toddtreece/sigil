module.exports = {
  extends: ['./.config/.eslintrc', 'prettier'],
  overrides: [
    {
      files: ['src/**/*.{ts,tsx}'],
      parserOptions: {
        tsconfigRootDir: __dirname,
      },
    },
  ],
};
