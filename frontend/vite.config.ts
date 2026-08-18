/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import mdx from "@mdx-js/rollup";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import rehypeSlug from "rehype-slug";

// https://vite.dev/config/
export default defineConfig({
  envPrefix: "PUBLIC_",
  plugins: [
    tanstackRouter({
      target: "react",
      routesDirectory: "./src/app/routes",
      generatedRouteTree: "./src/app/routeTree.gen.ts",
      autoCodeSplitting: true,
    }),
    mdx({ rehypePlugins: [rehypeSlug] }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@components": new URL("./src/components", import.meta.url).pathname,
      "@/lib": new URL("./src/lib", import.meta.url).pathname,
      "@": new URL("./src", import.meta.url).pathname,
    },
  },
  test: {
    environment: "node",
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "lcov"],
      include: [
        "src/config/**/*.ts",
        "src/lib/**/*.ts",
        "src/model/**/*.ts",
        "src/features/**/*.ts",
      ],
      exclude: [
        "src/**/*.test.{ts,tsx}",
        "src/**/index.ts",
        "src/**/*.interface.ts",
        "src/**/*.types.ts",
        "src/features/**/types/**/*.ts",
        "src/**/*.query.ts",
        "src/**/*.mutation.ts",
        "src/**/*.mutations.ts",
        "src/**/mock/**",
        "src/model/generated/**",
      ],
    },
  },
});
