import { describe, expect, it } from "vitest";

import { buildSpriteGenerationRequest } from "./sprite-generation-request";

describe("buildSpriteGenerationRequest", () => {
  it("maps a frame selection and reference image to an edit request", () => {
    expect(
      buildSpriteGenerationRequest("character", 12, {
        prompt: "Make the stride longer",
        creatingReference: {
          fileName: "stride.png",
          mimeType: "image/png",
          objectKey: "uploads/stride.png",
        },
        target: {
          nodeIds: ["walk"],
          frames: [
            { nodeId: "walk", index: 0 },
            { nodeId: "walk", index: 2 },
          ],
        },
      }),
    ).toEqual({
      assetId: 12,
      kind: "edit_frames",
      creative_brief: "Make the stride longer",
      targetAssetPaths: [
        "animations.walk.frames.0",
        "animations.walk.frames.2",
      ],
      parameters: {
        creating_reference: "uploads/stride.png",
        creating_reference_file_name: "stride.png",
        creating_reference_mime_type: "image/png",
      },
    });
  });

  it("maps an unselected object prompt to the prototype editor", () => {
    expect(
      buildSpriteGenerationRequest("object", 4, {
        prompt: "Add stronger highlights",
        target: { nodeIds: [], frames: [] },
      }),
    ).toEqual({
      assetId: 4,
      kind: "edit_object_prototype",
      creative_brief: "Add stronger highlights",
      targetAssetPaths: undefined,
      parameters: undefined,
    });
  });
});
