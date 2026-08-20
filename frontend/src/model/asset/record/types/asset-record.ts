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
export type AudioAssetKind = "audio";

export type SceneryCanvasDimensions = { width: number; height: number };

type AssetRecordBase<K extends AssetKind> = {
  mode: K;
  prompt: string;
};

export type SpriteAssetRecordData = {
  prototype: CharacterSpriteSheet;
  animations?: CharacterAnimation[];
  nodePositions: Record<string, AssetCanvasPosition>;
};

export type CharacterAssetRecord = AssetRecordBase<CharacterAssetKind> & {
  character: SpriteAssetRecordData;
};

export type ObjectAssetRecord = AssetRecordBase<ObjectAssetKind> & {
  object: SpriteAssetRecordData;
};

export type SceneryAssetRecord = AssetRecordBase<SceneryAssetKind> & {
  scenery: {
    layers: SceneryLayer[];
    dimensions?: SceneryCanvasDimensions;
  };
};

export type TilesetAssetRecord = AssetRecordBase<TilesetAssetKind> & {
  tileset: { gridSize: number; items: TilesetItem[] };
};

export type UISetAssetRecord = AssetRecordBase<UISetAssetKind> & {
  uiset: { components: UISetComponent[] };
};

export type AudioAssetRecord = AssetRecordBase<AudioAssetKind> & {
  audio: Record<string, never>;
};

type AssetRecordByKind = {
  character: CharacterAssetRecord;
  object: ObjectAssetRecord;
  scenery: SceneryAssetRecord;
  tileset: TilesetAssetRecord;
  uiset: UISetAssetRecord;
  audio: AudioAssetRecord;
};

export type AssetRecord = AssetRecordByKind[AssetKind];

export type AssetRecordForKind<K extends AssetKind> = AssetRecordByKind[K];
