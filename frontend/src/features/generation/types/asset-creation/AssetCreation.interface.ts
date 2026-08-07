import type { CreatableAssetKind, TilesetTile } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import type { Perspective } from "@/model/project";

type CommonAssetCreationDraft<K extends CreatableAssetKind> = {
  kind: K;
  name: string;
  prompt: string;
  canvasSize: string;
  useProjectContext: boolean;
};

export type VisualAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<
    Exclude<CreatableAssetKind, "audio" | "scenery" | "tileset" | "ui">
  > & {
    perspective: Perspective;
    directionCount: NonNullable<CreationRequest["directionCount"]>;
    reference: Reference | undefined;
  };

export type SceneryAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"scenery"> & {
    style: string;
    aspectRatio: string;
    layers: { description: string }[];
    reference: Reference | undefined;
  };

export type TilesetAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"tileset"> & {
    tiles: {
      description: string;
      reference: Reference | undefined;
      shape: TilesetTile[];
    }[];
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
  | SceneryAssetCreationDraft<Reference>
  | TilesetAssetCreationDraft<Reference>
  | UiAssetCreationDraft<Reference>
  | AudioAssetCreationDraft;
