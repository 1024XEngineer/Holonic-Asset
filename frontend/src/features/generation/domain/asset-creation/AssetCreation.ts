import {
  getDefaultAssetCanvasSize,
  type CreatableAssetKind,
} from "@/features/assets/domain";
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
    case "background":
      return {
        ...common,
        kind,
        backgroundType: "scenery",
        style: "",
        aspectRatio: "16:9",
        layers: [{ description: "" }],
        tiles: [{ description: "", reference: undefined }],
        reference: undefined,
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
    case "background":
      return {
        ...common,
        backgroundType: draft.backgroundType,
        style: draft.style,
        aspectRatio:
          draft.backgroundType === "scenery" ? draft.aspectRatio : undefined,
        layers: draft.backgroundType === "scenery" ? draft.layers : undefined,
        tiles: draft.backgroundType === "tiles" ? draft.tiles : undefined,
      };
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
