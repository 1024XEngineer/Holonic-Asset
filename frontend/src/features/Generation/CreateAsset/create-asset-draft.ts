import {
  getDefaultAssetCanvasSize,
  type CreatableAssetKind,
} from "@/types/asset-kind";
import type { CreationRequest } from "@/types/generation";

type CommonAssetCreationDraft<K extends CreatableAssetKind> = {
  kind: K;
  name: string;
  prompt: string;
  canvasSize: string;
  useProjectContext: boolean;
};

export type VisualAssetCreationDraft = CommonAssetCreationDraft<
  Exclude<CreatableAssetKind, "audio" | "background" | "ui">
> & {
  perspective: NonNullable<CreationRequest["perspective"]>;
  directionCount: NonNullable<CreationRequest["directionCount"]>;
  reference: File | undefined;
};

export type BackgroundAssetCreationDraft =
  CommonAssetCreationDraft<"background"> & {
    backgroundType: "scenery" | "tiles";
    style: string;
    aspectRatio: string;
    layers: { description: string }[];
    tiles: { description: string; reference: File | undefined }[];
    reference: File | undefined;
  };

export type UiAssetCreationDraft = CommonAssetCreationDraft<"ui"> & {
  style: string;
  reference: File | undefined;
  components: { name: string; description: string; isCustom: boolean }[];
};

export type AudioAssetCreationDraft = CommonAssetCreationDraft<"audio">;

export type CreateAssetDraft =
  | VisualAssetCreationDraft
  | BackgroundAssetCreationDraft
  | UiAssetCreationDraft
  | AudioAssetCreationDraft;

export function createAssetDraft(
  kind: CreatableAssetKind,
  initialPrompt = "",
): CreateAssetDraft {
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

export function toCreationRequest(draft: CreateAssetDraft): CreationRequest {
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
