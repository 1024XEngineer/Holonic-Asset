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

export type UiComponent = {
  id: string;
  label: string;
  kind: "panel" | "label" | "button";
  bounds: { x: number; y: number; width: number; height: number };
};

export type CharacterAssetKind = "character" | "object";
export type SceneryAssetKind = "scenery";
export type TilesetAssetKind = "tileset";
export type UiAssetKind = "ui";
export type AudioAssetKind = "audio";

export type CharacterAssetRecord = {
  mode: "character";
  prompt: string;
  character: {
    prototype: CharacterSpriteSheet;
    animations?: CharacterAnimation[];
    nodePositions: Record<string, AssetCanvasPosition>;
  };
};

export type SceneryAssetRecord = {
  mode: "scenery";
  prompt: string;
  scenery: { layers: SceneryLayer[] };
};

export type TilesetAssetRecord = {
  mode: "tileset";
  prompt: string;
  tileset: { gridSize: number; items: TilesetItem[] };
};

export type UiAssetRecord = {
  mode: "ui";
  prompt: string;
  ui: { components: UiComponent[] };
};

export type AudioAssetRecord = {
  mode: "audio";
  prompt: string;
  audio: Record<string, never>;
};

export type AssetRecord =
  | CharacterAssetRecord
  | SceneryAssetRecord
  | TilesetAssetRecord
  | UiAssetRecord
  | AudioAssetRecord;

export type AssetRecordForKind<K extends AssetKind> =
  K extends CharacterAssetKind
    ? CharacterAssetRecord
    : K extends SceneryAssetKind
      ? SceneryAssetRecord
      : K extends TilesetAssetKind
        ? TilesetAssetRecord
        : K extends UiAssetKind
          ? UiAssetRecord
          : AudioAssetRecord;
