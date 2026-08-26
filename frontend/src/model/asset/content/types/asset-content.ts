import type {
  AssetKind,
  CharacterAnimation,
  CharacterSpriteSheet,
  SceneryLayer,
} from "../../types";
import type { ItemTile } from "@/model/item-tile";

export type AssetCanvasPosition = { x: number; y: number };

export type TilesetItem = {
  id: string;
  label: string;
  /** Complete generated item image; tiles define its grid placement and selection footprint. */
  imageUrl?: string;
  /** Individual generated tile images, aligned with the tiles array. */
  tileUrls?: Array<string | undefined>;
  /** Every tileset tile occupied by this complete item, as [column, row]. */
  tiles: ItemTile[];
};

export type UISetComponent = {
  id: string;
  label: string;
  kind: "panel" | "label" | "button";
  bounds: { x: number; y: number; width: number; height: number };
};

export type CharacterAssetKind = "character";
type ObjectAssetKind = "object";
export type SceneryAssetKind = "scenery";
export type TilesetAssetKind = "tileset";
export type UISetAssetKind = "uiset";
export type AssetContentKind = Exclude<AssetKind, "audio">;
/** @deprecated Use AssetContentKind. */
export type AssetRecordKind = AssetContentKind;

export type SceneryCanvasDimensions = { width: number; height: number };

type AssetContentBase<K extends AssetContentKind> = {
  mode: K;
  prompt: string;
};

export type SpriteAssetRecordData = {
  prototype: CharacterSpriteSheet;
  animations?: CharacterAnimation[];
  nodePositions: Record<string, AssetCanvasPosition>;
};

export type CharacterAssetContent = AssetContentBase<CharacterAssetKind> & {
  character: SpriteAssetRecordData;
};

export type ObjectAssetContent = AssetContentBase<ObjectAssetKind> & {
  object: SpriteAssetRecordData;
};

export type SceneryAssetContent = AssetContentBase<SceneryAssetKind> & {
  scenery: {
    layers: SceneryLayer[];
    dimensions?: SceneryCanvasDimensions;
  };
};

export type TilesetAssetContent = AssetContentBase<TilesetAssetKind> & {
  tileset: { gridSize: number; items: TilesetItem[] };
};

export type UISetAssetContent = AssetContentBase<UISetAssetKind> & {
  uiset: {
    components: UISetComponent[];
    dimensions?: { width: number; height: number };
  };
};

type AssetContentByKind = {
  character: CharacterAssetContent;
  object: ObjectAssetContent;
  scenery: SceneryAssetContent;
  tileset: TilesetAssetContent;
  uiset: UISetAssetContent;
};

export type AssetContent = AssetContentByKind[AssetContentKind];

export type AssetContentForKind<K extends AssetContentKind> =
  AssetContentByKind[K];

export type CharacterAssetRecord = CharacterAssetContent;
export type ObjectAssetRecord = ObjectAssetContent;
export type SceneryAssetRecord = SceneryAssetContent;
export type TilesetAssetRecord = TilesetAssetContent;
export type UISetAssetRecord = UISetAssetContent;
export type AssetRecord = AssetContent;
export type AssetRecordForKind<K extends AssetContentKind> =
  AssetContentForKind<K>;
