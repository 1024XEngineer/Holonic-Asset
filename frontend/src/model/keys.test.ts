import { describe, expect, it } from "vitest";

import { audioKeys } from "./asset/audio/audio.keys";
import { assetKeys } from "./asset/library/keys";
import { recordKeys } from "./asset/record/record.keys";
import { quickGenerationKeys } from "./generation/quick/quick-generation.keys";
import { generationKeys } from "./generation/run/keys";
import { projectKeys } from "./project/keys";

describe("query keys", () => {
  it("builds stable, namespaced keys", () => {
    expect(projectKeys.list(7)).toEqual(["projects", 7, "list"]);
    expect(projectKeys.detail(7, "project-1")).toEqual([
      "projects",
      7,
      "detail",
      "project-1",
    ]);
    expect(assetKeys.library(7, "project-1")).toEqual([
      "assets",
      7,
      "library",
      "project-1",
    ]);
    expect(recordKeys.detail(7, "project-1", "asset-1")).toEqual([
      "record",
      7,
      "detail",
      "project-1",
      "asset-1",
    ]);
    expect(audioKeys.tracks(7)).toEqual(["audio", 7, "tracks"]);
    expect(generationKeys.runs(7, "project-1")).toEqual([
      "generation",
      7,
      "runs",
      "project-1",
    ]);
    expect(quickGenerationKeys.assets(7)).toEqual([
      "quick-generation",
      7,
      "assets",
    ]);
  });
});
