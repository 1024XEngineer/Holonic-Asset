import { renderToStaticMarkup } from "react-dom/server";
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
    const html = renderToStaticMarkup(
      withI18n(<GenerationQueue runs={[generationRun("failed")]} />),
    );

    expect(html).not.toContain("animate-spin");
  });

  it("shows a loading animation while a run is active", () => {
    const html = renderToStaticMarkup(
      withI18n(<GenerationQueue runs={[generationRun("processing")]} />),
    );

    expect(html).toContain("animate-spin");
  });

  it.each([
    ["pending", "Pending"],
    ["processing", "Processing"],
    ["failed", "Failed"],
  ] as const)("localizes the %s status", (status, label) => {
    const html = renderToStaticMarkup(
      withI18n(<GenerationQueue runs={[generationRun(status)]} />),
    );

    expect(html).toContain(label);
    expect(html).not.toContain(`>${status}<`);
  });
});
