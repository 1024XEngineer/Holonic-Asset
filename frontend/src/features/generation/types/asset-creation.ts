import type { CreatableAssetKind } from "@/model/asset";
import type { Perspective } from "@/model/project";
import type { ItemTile } from "@/model/item-tile";
import type { SceneryAspectRatio } from "../create-asset/scenery-aspect-ratio";

type CommonAssetCreationDraft<K extends CreatableAssetKind> = {
  kind: K;
  name: string;
  prompt: string;
  canvasSize: string;
};

export type VisualAssetCreationDraft<CreatingReference = unknown> =
  CommonAssetCreationDraft<
    Exclude<CreatableAssetKind, "audio" | "scenery" | "tileset" | "uiset">
  > & {
    perspective: Perspective;
    creatingReference: CreatingReference | undefined;
  };

export type SceneryAssetCreationDraft<CreatingReference = unknown> =
  CommonAssetCreationDraft<"scenery"> & {
    aspectRatio: SceneryAspectRatio;
    creatingReference: CreatingReference | undefined;
  };

export type TilesetAssetCreationDraft = CommonAssetCreationDraft<"tileset"> & {
  tiles: {
    name: string;
    description: string;
    shape: ItemTile[];
  }[];
};

export type UISetAssetCreationDraft<CreatingReference = unknown> =
  CommonAssetCreationDraft<"uiset"> & {
    dimensions: { width: number; height: number };
    style: string;
    creatingReference: CreatingReference | undefined;
    components: { id: string; name: string; description: string }[];
  };

export type AudioAssetCreationDraft = CommonAssetCreationDraft<"audio">;

export type AssetCreationDraft<CreatingReference = unknown> =
  | VisualAssetCreationDraft<CreatingReference>
  | SceneryAssetCreationDraft<CreatingReference>
  | TilesetAssetCreationDraft
  | UISetAssetCreationDraft<CreatingReference>
  | AudioAssetCreationDraft;
