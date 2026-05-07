import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";

export default [
    js.configs.recommended,
    ...svelte.configs["flat/recommended"],
    {
        languageOptions: {
            ecmaVersion: 2022,
            sourceType: "module",
            globals: {
                window: "readonly",
                document: "readonly",
                fetch: "readonly",
                console: "readonly",
                setTimeout: "readonly",
                clearTimeout: "readonly",
                setInterval: "readonly",
                clearInterval: "readonly",
                URL: "readonly",
                URLSearchParams: "readonly",
                localStorage: "readonly",
                sessionStorage: "readonly",
                ResizeObserver: "readonly",
                EventSource: "readonly",
            },
        },
        rules: {
            "no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],
        },
    },
    {
        ignores: [
            ".svelte-kit/**",
            "build/**",
            "node_modules/**",
        ],
    },
];
