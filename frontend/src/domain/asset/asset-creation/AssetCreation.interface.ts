import type { CreatableAssetKind } from "@/domain/asset";
import type { CreationRequest } from "@/domain/generation";

type CommonAssetCreationDraft<K extends CreatableAssetKind> = {
  kind: K;
  name: string;
  prompt: string;
  canvasSize: string;
  useProjectContext: boolean;
};

export type VisualAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<
    Exclude<CreatableAssetKind, "audio" | "background" | "ui">
  > & {
    perspective: NonNullable<CreationRequest["perspective"]>;
    directionCount: NonNullable<CreationRequest["directionCount"]>;
    reference: Reference | undefined;
  };

export type BackgroundAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"background"> & {
    backgroundType: "scenery" | "tiles";
    style: string;
    aspectRatio: string;
    layers: { description: string }[];
    tiles: { description: string; reference: Reference | undefined }[];
    reference: Reference | undefined;
  };

export type UiAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"ui"> & {
    style: string;
    reference: Reference | undefined;
    components: { name: string; description: string; isCustom: boolean }[];
  };

export type AudioAssetCreationDraft = CommonAssetCreationDraft<"audio">;

export type AssetCreationDraft<Reference = unknown> =
  | VisualAssetCreationDraft<Reference>
  | BackgroundAssetCreationDraft<Reference>
  | UiAssetCreationDraft<Reference>
  | AudioAssetCreationDraft;
