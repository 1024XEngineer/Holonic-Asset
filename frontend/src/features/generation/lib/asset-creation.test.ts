import { describe, expect, it } from "vitest";

import type { CreatableAssetKind } from "@/model/asset";

import {
  assetCreationDraftSchema,
  createAssetCreationDraft,
  getAssetCreationValidationMessageKey,
  toCreationRequest,
} from "./asset-creation";

const kinds: CreatableAssetKind[] = [
  "character",
  "object",
  "scenery",
  "tileset",
  "uiset",
  "audio",
];

describe("asset creation", () => {
  it.each(kinds)("creates defaults for %s assets", (kind) => {
    const draft = createAssetCreationDraft(kind, "  Initial prompt  ");

    expect(draft).toMatchObject({
      kind,
      name: "",
      prompt: "Initial prompt",
    });
    expect(draft.canvasSize).toMatch(/px$/);
  });

  it("creates kind-specific request payloads", () => {
    const reference = { id: "reference" };
    const drafts = kinds.map((kind) => {
      const draft = createAssetCreationDraft<typeof reference>(kind);
      draft.name = "  Asset name  ";
      draft.prompt = "  Asset prompt  ";
      if ("reference" in draft) draft.reference = reference;
      return draft;
    });

    const requests = drafts.map(toCreationRequest);

    expect(requests).toEqual([
      expect.objectContaining({
        kind: "character",
        name: "Asset name",
        prompt: "Asset prompt",
        perspective: "Top-Down",
        reference,
      }),
      expect.objectContaining({
        kind: "object",
        perspective: "Top-Down",
        reference,
      }),
      expect.objectContaining({
        kind: "scenery",
        style: "",
        aspectRatio: "16:9",
        layers: [{ description: "" }],
      }),
      expect.objectContaining({
        kind: "tileset",
        tiles: [{ description: "", reference: undefined, shape: [[0, 0]] }],
      }),
      expect.objectContaining({
        kind: "uiset",
        reference,
        canvasSize: "1024 x 768 px",
        dimensions: { width: 1024, height: 768 },
        components: [{ name: "", description: "" }],
      }),
      expect.objectContaining({ kind: "audio" }),
    ]);
    expect(requests[0]).not.toHaveProperty("directionCount");
    expect(requests[1]).not.toHaveProperty("directionCount");
  });

  it("derives the UI Set canvas size from its dimensions", () => {
    const draft = createAssetCreationDraft("uiset");
    if (draft.kind !== "uiset") throw new Error("Expected a UI Set draft.");

    draft.dimensions = { width: 1280, height: 720 };

    expect(toCreationRequest(draft)).toMatchObject({
      canvasSize: "1280 x 720 px",
      dimensions: { width: 1280, height: 720 },
    });
    expect(toCreationRequest(draft).components?.[0]).not.toHaveProperty("id");
  });

  it.each(["0 × 0 px", "0 × 32 px", "32 × 0 px", "large"])(
    "rejects invalid user-entered canvas size: %s",
    (canvasSize) => {
      const draft = createAssetCreationDraft("character");
      const result = assetCreationDraftSchema.safeParse({
        ...draft,
        name: "Asset name",
        prompt: "Asset prompt",
        canvasSize,
      });

      expect(result.success).toBe(false);
    },
  );

  it.each([
    ["UI Set style is required.", "validation.uiSetStyleRequired"],
    ["Every component needs a name.", "validation.uiSetComponentNameRequired"],
    [
      "Every component needs a description.",
      "validation.uiSetComponentDescriptionRequired",
    ],
    [
      "Select a supported canvas width.",
      "validation.uiSetCanvasWidthUnsupported",
    ],
    [
      "Select a supported canvas height.",
      "validation.uiSetCanvasHeightUnsupported",
    ],
  ])("maps UI Set validation message %s to %s", (message, key) => {
    expect(getAssetCreationValidationMessageKey(message)).toBe(key);
  });
});
