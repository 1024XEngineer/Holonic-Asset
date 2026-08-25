import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import type { GenerationRun } from "@/model/generation";
import { withI18n } from "@/testing/with-i18n";

import { GenerationQueue } from "./generation-queue";

function generationRun(status: GenerationRun["status"]): GenerationRun {
  return {
    id: `run-${status}`,
    projectId: "moonlit-orchard",
    status,
    kind: "character",
    name: "Swordsman",
    prompt: "Four-direction top-down swordsman",
    canvasSize: "64 × 64 px",
  };
}

describe("GenerationQueue", () => {
  it("does not show a loading animation when every run has failed", () => {
    const html = renderQueue([generationRun("failed")]);

    expect(html).not.toContain("animate-spin");
  });

  it("shows a loading animation while a run is active", () => {
    const html = renderQueue([generationRun("processing")]);

    expect(html).toContain("animate-spin");
  });

  it("offers retry and delete actions only for failed runs", () => {
    const failedHtml = renderQueue([generationRun("failed")]);
    const processingHtml = renderQueue([generationRun("processing")]);

    expect(failedHtml).toContain('aria-label="Retry Swordsman"');
    expect(failedHtml).toContain('aria-label="Delete Swordsman"');
    expect(processingHtml).not.toContain('aria-label="Retry Swordsman"');
    expect(processingHtml).not.toContain('aria-label="Delete Swordsman"');
  });

  it("does not repeat the asset type beside the status", () => {
    const html = renderQueue([generationRun("failed")]);

    expect(html).not.toContain(">Character<");
  });

  it.each([
    ["pending", "Pending"],
    ["processing", "Processing"],
    ["failed", "Failed"],
  ] as const)("localizes the %s status", (status, label) => {
    const html = renderQueue([generationRun(status)]);

    expect(html).toContain(label);
    expect(html).not.toContain(`>${status}<`);
  });
});

function renderQueue(runs: GenerationRun[]) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return renderToStaticMarkup(
    withI18n(
      <QueryClientProvider client={queryClient}>
        <GenerationQueue runs={runs} />
      </QueryClientProvider>,
    ),
  );
}
