const {
    defineConfig,
    globalIgnores,
} = require("eslint/config");

const globals = require("globals");
const js = require("@eslint/js");

const {
    FlatCompat,
} = require("@eslint/eslintrc");

const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all
});

module.exports = defineConfig([{
    extends: compat.extends("eslint:recommended"),

    languageOptions: {
        ecmaVersion: 2022,
        sourceType: "module",
        parserOptions: {},

        globals: {
            ...globals.browser,
        },
    },

    rules: {
        "linebreak-style": "off",
    },
}, globalIgnores(["**/node_modules"])]);
