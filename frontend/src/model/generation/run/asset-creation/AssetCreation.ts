import {
  getDefaultAssetCanvasSize,
  type CreatableAssetKind,
} from "../../../asset";
import type { CreationRequest } from "../generation";

import type { AssetCreationDraft } from "./AssetCreation.interface";

export function createAssetCreationDraft<Reference = unknown>(
  kind: CreatableAssetKind,
  initialPrompt = "",
): AssetCreationDraft<Reference> {
  const common = {
    name: "",
    prompt: initialPrompt.trim(),
    canvasSize: getDefaultAssetCanvasSize(kind),
    useProjectContext: true,
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
        tiles: [{ description: "", reference: undefined }],
      };
    case "ui":
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
        perspective: "top-down",
        directionCount: "4",
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
    useProjectContext: draft.useProjectContext,
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
    case "ui":
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
            directionCount: draft.directionCount,
            reference: draft.reference,
          }
        : common;
  }
}
