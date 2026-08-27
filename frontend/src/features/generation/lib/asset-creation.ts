import type { CreatableAssetKind } from "@/model/asset";
import { assetCanvasSizeSchema, assetTagSchema } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import { getDefaultAssetCanvasSize } from "@/model";
import {
  perspectiveOptions,
  perspectiveSchema,
  type Perspective,
} from "@/model/project";
import { z } from "zod";

import {
  defaultUISetCanvasDimensions,
  isUISetCanvasHeight,
  isUISetCanvasWidth,
} from "../create-asset/ui-set-canvas";
import {
  defaultSceneryAspectRatio,
  getSceneryCanvasSize,
  getSceneryDimensions,
  sceneryAspectRatios,
} from "../create-asset/scenery-aspect-ratio";
import type { AssetCreationDraft } from "../types";

const commonAssetCreationDraftShape = {
  name: z.string().trim().min(1, "Asset name is required."),
  prompt: z.string().trim().min(1, "Creative brief is required."),
  canvasSize: assetCanvasSizeSchema,
};

const itemTileSchema = z.tuple([z.number(), z.number()]);

export function createUISetComponent() {
  return { id: crypto.randomUUID(), name: "", description: "" };
}

function formatCanvasSize({
  width,
  height,
}: {
  width: number;
  height: number;
}) {
  return `${width} x ${height} px`;
}

export const assetCreationDraftSchema = z.discriminatedUnion("kind", [
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("audio"),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("scenery"),
    aspectRatio: z.enum(sceneryAspectRatios),
    creatingReference: z.unknown().optional(),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("tileset"),
    tiles: z.array(
      z.object({
        name: z.string().trim().min(1, "Each tileset item needs a name."),
        description: z
          .string()
          .trim()
          .min(1, "Each tileset item needs a description."),
        shape: z
          .array(itemTileSchema)
          .min(1, "Each tileset item must have at least one occupied tile."),
      }),
    ),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.literal("uiset"),
    dimensions: z.object({
      width: z
        .number()
        .int()
        .refine(isUISetCanvasWidth, "Select a supported canvas width."),
      height: z
        .number()
        .int()
        .refine(isUISetCanvasHeight, "Select a supported canvas height."),
    }),
    style: z.string().trim().min(1, "UI Set style is required."),
    creatingReference: z.unknown().optional(),
    components: z
      .array(
        z.object({
          id: z.string(),
          name: z.string().trim().min(1, "Every component needs a name."),
          description: z
            .string()
            .trim()
            .min(1, "Every component needs a description."),
        }),
      )
      .min(1, "Add at least one UI Set component."),
  }),
  z.object({
    ...commonAssetCreationDraftShape,
    kind: z.enum(["character", "object"]),
    perspective: perspectiveSchema,
    tags: z.array(assetTagSchema),
    creatingReference: z.unknown().optional(),
  }),
]);

export function createAssetCreationDraft<CreatingReference = unknown>(
  kind: CreatableAssetKind,
  initialPrompt = "",
  perspective: Perspective = perspectiveOptions[0],
): AssetCreationDraft<CreatingReference> {
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
        aspectRatio: defaultSceneryAspectRatio,
        canvasSize: getSceneryCanvasSize(defaultSceneryAspectRatio),
        creatingReference: undefined,
      };
    case "tileset":
      return {
        ...common,
        kind,
        tiles: [{ name: "", description: "", shape: [[0, 0]] }],
      };
    case "uiset":
      const dimensions = { ...defaultUISetCanvasDimensions };
      return {
        ...common,
        kind,
        canvasSize: formatCanvasSize(dimensions),
        dimensions,
        style: "",
        creatingReference: undefined,
        components: [createUISetComponent()],
      };
    default:
      return {
        ...common,
        kind,
        perspective,
        tags: [],
        creatingReference: undefined,
      };
  }
}

export function toCreationRequest<CreatingReference>(
  draft: AssetCreationDraft<CreatingReference>,
): CreationRequest<CreatingReference> {
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
        canvasSize: getSceneryCanvasSize(draft.aspectRatio),
        dimensions: getSceneryDimensions(draft.aspectRatio),
        creatingReference: draft.creatingReference,
      };
    case "tileset":
      return {
        ...common,
        tiles: draft.tiles.map((tile) => ({
          ...tile,
          name: tile.name.trim(),
          description: tile.description.trim(),
        })),
      };
    case "uiset":
      return {
        ...common,
        canvasSize: formatCanvasSize(draft.dimensions),
        dimensions: draft.dimensions,
        style: draft.style,
        creatingReference: draft.creatingReference,
        components: draft.components.map(
          ({ id: _, ...component }) => component,
        ),
      };
    default:
      return draft.kind === "character" || draft.kind === "object"
        ? {
            ...common,
            perspective: draft.perspective,
            tags: draft.tags,
            creatingReference: draft.creatingReference,
          }
        : common;
  }
}
