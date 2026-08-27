import { describe, expect, it } from "vitest";

import type { CreatableAssetKind } from "@/model/asset";

import {
  assetCreationDraftSchema,
  createAssetCreationDraft,
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
    const creatingReference = { id: "creating-reference" };
    const drafts = kinds.map((kind) => {
      const draft = createAssetCreationDraft<typeof creatingReference>(kind);
      draft.name = "  Asset name  ";
      draft.prompt = "  Asset prompt  ";
      if ("creatingReference" in draft)
        draft.creatingReference = creatingReference;
      return draft;
    });

    const requests = drafts.map(toCreationRequest);

    expect(requests).toEqual([
      expect.objectContaining({
        kind: "character",
        name: "Asset name",
        prompt: "Asset prompt",
        perspective: "Top-Down",
        creatingReference,
      }),
      expect.objectContaining({
        kind: "object",
        perspective: "Top-Down",
        creatingReference,
      }),
      expect.objectContaining({
        kind: "scenery",
        name: "Asset name",
        prompt: "Asset prompt",
      }),
      expect.objectContaining({
        kind: "tileset",
        tiles: [{ name: "", description: "", shape: [[0, 0]] }],
      }),
      expect.objectContaining({
        kind: "uiset",
        creatingReference,
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

  it.each([
    ["16:9", "1536 × 1024 px"],
    ["4:3", "1024 × 768 px"],
    ["21:9", "1792 × 768 px"],
    ["1:1", "1024 × 1024 px"],
    ["3:2", "1536 × 1024 px"],
    ["9:16", "1024 × 1536 px"],
    ["2:3", "1024 × 1536 px"],
  ] as const)(
    "maps scenery aspect ratio %s to canvas size %s",
    (aspectRatio, canvasSize) => {
      const draft = createAssetCreationDraft("scenery");
      if (draft.kind !== "scenery") throw new Error("Expected scenery draft.");

      draft.aspectRatio = aspectRatio;
      draft.canvasSize = "1 × 1 px";

      expect(toCreationRequest(draft)).toMatchObject({
        canvasSize,
        dimensions: sceneryDimensions[aspectRatio],
      });
    },
  );

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

  it("inherits custom project perspective for visual assets", () => {
    const draft = createAssetCreationDraft("character", "", "Side-On");
    expect(draft).toMatchObject({
      kind: "character",
      perspective: "Side-On",
    });
  });

  it("trims tileset item metadata", () => {
    const draft = createAssetCreationDraft("tileset");
    if (draft.kind !== "tileset") throw new Error("expected tileset draft");
    draft.tiles = [
      {
        name: "  Grass edge  ",
        description: "  A seamless grass edge  ",
        shape: [[0, 0]],
      },
    ];

    expect(toCreationRequest(draft).tiles).toEqual([
      {
        name: "Grass edge",
        description: "A seamless grass edge",
        shape: [[0, 0]],
      },
    ]);
  });
});

const sceneryDimensions = {
  "16:9": { width: 1536, height: 1024 },
  "4:3": { width: 1024, height: 768 },
  "21:9": { width: 1792, height: 768 },
  "1:1": { width: 1024, height: 1024 },
  "3:2": { width: 1536, height: 1024 },
  "9:16": { width: 1024, height: 1536 },
  "2:3": { width: 1024, height: 1536 },
} as const;
