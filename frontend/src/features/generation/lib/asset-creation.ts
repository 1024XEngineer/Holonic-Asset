import type { CreatableAssetKind } from "@/model/asset";
import { assetCanvasSizeSchema } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import { getDefaultAssetCanvasSize } from "@/model";
import { perspectiveOptions, perspectiveSchema } from "@/model/project";
import { z } from "zod";

import type { AssetCreationDraft } from "../types";

const commonAssetCreationDraftShape = {
  name: z.string().trim().min(1, "Asset name is required."),
  prompt: z.string().trim().min(1, "Creative brief is required."),
  canvasSize: assetCanvasSizeSchema,
};

const itemTileSchema = z.tuple([z.number(), z.number()]);

export const assetCreationDraftSchema = z.discriminatedUnion("kind", [
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("audio"),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("scenery"),
    style: z.string(),
    aspectRatio: z.string(),
    layers: z.array(z.object({ description: z.string() })),
    reference: z.unknown().optional(),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("tileset"),
    tiles: z.array(
      z.object({
        description: z.string(),
        reference: z.unknown().optional(),
        shape: z
          .array(itemTileSchema)
          .min(1, "Each tileset item must have at least one occupied tile."),
      }),
    ),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("uiset"),
    style: z.string(),
    reference: z.unknown().optional(),
    components: z.array(
      z.object({
        name: z.string(),
        description: z.string(),
        isCustom: z.boolean(),
      }),
    ),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.enum(["character", "object"]),
    perspective: perspectiveSchema,
    reference: z.unknown().optional(),
  }),
]);

export function createAssetCreationDraft<Reference = unknown>(
  kind: CreatableAssetKind,
  initialPrompt = "",
): AssetCreationDraft<Reference> {
  const common = {
    name: "",
    prompt: initialPrompt.trim(),
    canvasSize: getDefaultAssetCanvasSize(kind),
  };

  switch (kind) {
    case "audio":
      return { ...common, kind };
    case "scenery":
      return {
        ...common,
        kind,
        style: "",
        aspectRatio: "16:9",
        layers: [{ description: "" }],
        reference: undefined,
      };
    case "tileset":
      return {
        ...common,
        kind,
        tiles: [{ description: "", reference: undefined, shape: [[0, 0]] }],
      };
    case "uiset":
      return {
        ...common,
        kind,
        style: "",
        reference: undefined,
        components: [{ name: "", description: "", isCustom: false }],
      };
    default:
      return {
        ...common,
        kind,
        perspective: perspectiveOptions[0],
        reference: undefined,
      };
  }
}

export function toCreationRequest<Reference>(
  draft: AssetCreationDraft<Reference>,
): CreationRequest<Reference> {
  const common = {
    kind: draft.kind,
    name: draft.name.trim(),
    prompt: draft.prompt.trim(),
    canvasSize: draft.canvasSize,
  };

  switch (draft.kind) {
    case "audio":
      return common;
    case "scenery":
      return {
        ...common,
        style: draft.style,
        aspectRatio: draft.aspectRatio,
        layers: draft.layers,
      };
    case "tileset":
      return { ...common, tiles: draft.tiles };
    case "uiset":
      return {
        ...common,
        style: draft.style,
        reference: draft.reference,
        components: draft.components,
      };
    default:
      return draft.kind === "character" || draft.kind === "object"
        ? {
            ...common,
            perspective: draft.perspective,
            reference: draft.reference,
          }
        : common;
  }
}
