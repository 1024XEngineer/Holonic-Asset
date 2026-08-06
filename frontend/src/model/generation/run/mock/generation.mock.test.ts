import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { resetMockProjects } from "@/model/project/mock/project.mock";
import type { GenerationRun } from "../types";
import { completeMockGeneration } from "./generation.mock";

describe("completeMockGeneration", () => {
  beforeEach(() => {
    resetMockProjects();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("preserves an explicitly requested Isometric perspective", async () => {
    const result = await complete(generationRun({ perspective: "Isometric" }));

    expect(result.asset.perspective).toBe("Isometric");
  });

  it("inherits the project perspective when the request omits it", async () => {
    const result = await complete(
      generationRun({ kind: "ui", projectId: "iron-harbor" }),
    );

    expect(result.asset.perspective).toBe("Side-On");
  });
});

function generationRun(overrides: Partial<GenerationRun> = {}): GenerationRun {
  return {
    id: "run-1",
    projectId: "moonlit-orchard",
    status: "processing",
    kind: "character",
    name: "Swordsman",
    prompt: "Four-direction swordsman",
    canvasSize: "64 x 64 px",
    useProjectContext: true,
    ...overrides,
  };
}

async function complete(run: GenerationRun) {
  const completion = completeMockGeneration(run);
  await vi.advanceTimersByTimeAsync(700);
  return completion;
}
