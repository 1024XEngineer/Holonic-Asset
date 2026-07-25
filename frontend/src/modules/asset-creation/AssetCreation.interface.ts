import type { CreatableAssetKind } from "@/types/asset-kind";
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

export type AssetCreationDraft =
  | VisualAssetCreationDraft
  | BackgroundAssetCreationDraft
  | UiAssetCreationDraft
  | AudioAssetCreationDraft;
