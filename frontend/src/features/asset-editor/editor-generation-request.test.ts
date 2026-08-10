import { describe, expect, it } from "vitest";

import { buildInspectorGenerationRequest } from "./editor-generation-request";

describe("buildInspectorGenerationRequest", () => {
  it("maps a frame selection and reference image to an edit request", () => {
    expect(
      buildInspectorGenerationRequest("character", 12, {
        prompt: "Make the stride longer",
        reference: {
          fileName: "stride.png",
          mimeType: "image/png",
          dataUrl: "data:image/png;base64,c3RyaWRl",
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
      kind: "edit_character_frames",
      prompt: "Make the stride longer",
      targetAssetPaths: [
        "animations.walk.frames.0",
        "animations.walk.frames.2",
      ],
      parameters: {
        reference: "data:image/png;base64,c3RyaWRl",
        referenceFileName: "stride.png",
        referenceMimeType: "image/png",
      },
    });
  });

  it("maps an unselected object prompt to the prototype editor", () => {
    expect(
      buildInspectorGenerationRequest("object", 4, {
        prompt: "Add stronger highlights",
        target: { nodeIds: [], frames: [] },
      }),
    ).toEqual({
      assetId: 4,
      kind: "edit_object_prototype",
      prompt: "Add stronger highlights",
      targetAssetPaths: undefined,
      parameters: undefined,
    });
  });
});
