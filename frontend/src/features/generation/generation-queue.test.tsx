import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { GenerationRun } from "@/model/generation";

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
    useProjectContext: true,
  };
}

describe("GenerationQueue", () => {
  it("does not show a loading animation when every run has failed", () => {
    const html = renderToStaticMarkup(
      <GenerationQueue runs={[generationRun("failed")]} />,
    );

    expect(html).not.toContain("animate-spin");
  });

  it("shows a loading animation while a run is active", () => {
    const html = renderToStaticMarkup(
      <GenerationQueue runs={[generationRun("processing")]} />,
    );

    expect(html).toContain("animate-spin");
  });
});
