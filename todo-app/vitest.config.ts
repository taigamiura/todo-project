import { configDefaults, defineConfig } from "vitest/config";

const unitTestInclude = ["middleware.test.ts", "src/**/*.test.{ts,tsx}"];
const contractTestInclude = ["src/**/*.contract.test.ts"];
const baseExclude = [...configDefaults.exclude, "tests/e2e/**"];

const coverage = {
  provider: "v8" as const,
  reporter: ["text", "json-summary"],
  all: true,
  include: ["src/**/*.{ts,tsx}", "middleware.ts"],
  exclude: ["src/**/*.d.ts", "src/**/types/*.ts", "src/lib/api/generated/**", "src/test/**"],
  lines: 100,
  functions: 100,
  branches: 100,
  statements: 100,
};

const contractTestsEnabled = process.env.ENABLE_CONTRACT_TESTS === "1";

export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    include: contractTestsEnabled ? [...unitTestInclude, ...contractTestInclude] : unitTestInclude,
    exclude: contractTestsEnabled ? baseExclude : [...baseExclude, ...contractTestInclude],
    coverage,
  },
});