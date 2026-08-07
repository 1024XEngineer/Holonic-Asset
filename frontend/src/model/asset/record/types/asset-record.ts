import type {
  AssetKind,
  CharacterAnimation,
  CharacterSpriteSheet,
  SceneryLayer,
} from "../../types";

export type AssetCanvasPosition = { x: number; y: number };

/** Global [column, row] coordinate in the tileset grid. */
export type TilesetCell = [column: number, row: number];

export type TilesetItem = {
  id: string;
  label: string;
  /** Complete generated item image; tiles are only a front-end interaction map. */
  imageUrl?: string;
  /** Every tileset tile occupied by this complete item, as [column, row]. */
  tiles: TilesetCell[];
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
  scenery: { layers: SceneryLayer[] };
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
