import type { CreatableAssetKind } from "@/model/asset";
import type { Perspective } from "@/model/project";
import type { ItemTile } from "@/model/item-tile";

type CommonAssetCreationDraft<K extends CreatableAssetKind> = {
  kind: K;
  name: string;
  prompt: string;
  canvasSize: string;
};

export type VisualAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<
    Exclude<CreatableAssetKind, "audio" | "scenery" | "tileset" | "uiset">
  > & {
    perspective: Perspective;
    reference: Reference | undefined;
  };

export type SceneryAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"scenery"> & {
    style: string;
    aspectRatio: string;
    layers: { description: string }[];
    reference: Reference | undefined;
  };

export type TilesetAssetCreationDraft = CommonAssetCreationDraft<"tileset"> & {
  tiles: {
    name: string;
    description: string;
    shape: ItemTile[];
  }[];
};

export type UISetAssetCreationDraft<Reference = unknown> =
  CommonAssetCreationDraft<"uiset"> & {
    style: string;
    reference: Reference | undefined;
    components: { name: string; description: string; isCustom: boolean }[];
  };

export type AudioAssetCreationDraft = CommonAssetCreationDraft<"audio">;

export type AssetCreationDraft<Reference = unknown> =
  | VisualAssetCreationDraft<Reference>
  | SceneryAssetCreationDraft<Reference>
  | TilesetAssetCreationDraft
  | UISetAssetCreationDraft<Reference>
  | AudioAssetCreationDraft;
