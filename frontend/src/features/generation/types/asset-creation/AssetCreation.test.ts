import { describe, expect, it } from "vitest";

import type { CreatableAssetKind } from "@/model/asset";

import { createAssetCreationDraft, toCreationRequest } from "./AssetCreation";

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
      useProjectContext: true,
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

    expect(drafts.map(toCreationRequest)).toEqual([
      expect.objectContaining({
        kind: "character",
        name: "Asset name",
        prompt: "Asset prompt",
        perspective: "Top-Down",
        directionCount: "4",
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
        components: [{ name: "", description: "", isCustom: false }],
      }),
      expect.objectContaining({ kind: "audio" }),
    ]);
  });
});
