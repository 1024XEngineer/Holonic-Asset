import { defineConfig } from "vitest/config";

export default defineConfig({
  envPrefix: "PUBLIC_",
  resolve: {
    alias: {
      "@": new URL("./src", import.meta.url).pathname,
    },
  },
  test: {
    environment: "node",
    include: ["tests/integration/**/*.integration.ts"],
    testTimeout: 30_000,
  },
});
