import { describe, expect, it } from "vitest";

import { audioKeys } from "./asset/audio/audio.keys";
import { assetKeys } from "./asset/library/keys";
import { recordKeys } from "./asset/record/record.keys";
import { quickGenerationKeys } from "./generation/quick/quick-generation.keys";
import { generationKeys } from "./generation/run/keys";
import { projectKeys } from "./project/keys";

describe("query keys", () => {
  it("builds stable, namespaced keys", () => {
    expect(projectKeys.list()).toEqual(["projects", "list"]);
    expect(assetKeys.library("project-1")).toEqual([
      "assets",
      "library",
      "project-1",
    ]);
    expect(recordKeys.detail("project-1", "asset-1")).toEqual([
      "record",
      "detail",
      "project-1",
      "asset-1",
    ]);
    expect(audioKeys.tracks()).toEqual(["audio", "tracks"]);
    expect(generationKeys.runs("project-1")).toEqual([
      "generation",
      "runs",
      "project-1",
    ]);
    expect(quickGenerationKeys.assets()).toEqual([
      "quick-generation",
      "assets",
    ]);
  });
});
