import { describe, expect, it } from "vitest";

import type { GenerationRun } from "./types";
import {
  findSettledGenerationRunIds,
  GENERATION_POLL_INTERVAL_MS,
  generationPollingInterval,
  isGenerationRunActive,
} from "./generation-polling";

describe("generation polling", () => {
  it("polls while pending or processing runs exist", () => {
    expect(isGenerationRunActive(run("pending", "pending"))).toBe(true);
    expect(isGenerationRunActive(run("processing", "processing"))).toBe(true);
    expect(isGenerationRunActive(run("ready", "awaiting_application"))).toBe(
      false,
    );
    expect(isGenerationRunActive(run("failed", "failed"))).toBe(false);
    expect(generationPollingInterval(undefined)).toBe(false);
    expect(generationPollingInterval([run("failed", "failed")])).toBe(false);
    expect(generationPollingInterval([run("pending", "pending")])).toBe(
      GENERATION_POLL_INTERVAL_MS,
    );
  });

  it("finds runs that left the active queue", () => {
    expect(
      findSettledGenerationRunIds(
        [run("one", "pending"), run("two", "processing"), run("old", "failed")],
        [run("two", "processing"), run("three", "pending")],
      ),
    ).toEqual(["one"]);

    expect(
      findSettledGenerationRunIds(
        [run("ready", "processing")],
        [run("ready", "awaiting_application")],
      ),
    ).toEqual([]);
  });
});

function run(id: string, status: GenerationRun["status"]): GenerationRun {
  return {
    id,
    projectId: "project-1",
    status,
    kind: "character",
    name: "Hero",
    prompt: "Hero",
    canvasSize: "64 × 64 px",
  };
}
