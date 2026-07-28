import type { AssetKind } from "../library";

export type EditorCanvasPosition = { x: number; y: number };

export type EditorCharacterSpriteSheet = {
  format: "png-sprite-sheet";
  imageUrl: string;
  frameWidth: number;
  frameHeight: number;
  columns: number;
  rows: number;
  row?: number;
};

export type EditorCharacterAnimationClip = {
  kind: "clip";
  id: string;
  label: string;
  frameCount: number;
  spriteSheet?: EditorCharacterSpriteSheet;
  audio?: { label: string; time: string };
};

export type EditorCharacterAnimationGroup = {
  kind: "group";
  id: string;
  label: string;
  directions: EditorCharacterAnimationClip[];
};

export type EditorCharacterAnimation =
  | EditorCharacterAnimationClip
  | EditorCharacterAnimationGroup;

export type EditorSceneryLayer = {
  id: string;
  label: string;
  detail: string;
  imageUrl: string;
  blendMode: "normal" | "multiply";
};

/** Global [column, row] coordinate in the tileset grid. */
export type EditorTilesetCell = [column: number, row: number];

export type EditorTilesetItem = {
  id: string;
  label: string;
  /** Complete generated item image; tiles are only a front-end interaction map. */
  imageUrl?: string;
  /** Every tileset tile occupied by this complete item, as [column, row]. */
  tiles: EditorTilesetCell[];
};

export type EditorUiComponent = {
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

export type CharacterEditorRecord = {
  mode: "character";
  prompt: string;
  character: {
    prototype: EditorCharacterSpriteSheet;
    animations?: EditorCharacterAnimation[];
    nodePositions: Record<string, EditorCanvasPosition>;
  };
};

export type SceneryEditorRecord = {
  mode: "scenery";
  prompt: string;
  scenery: { layers: EditorSceneryLayer[] };
};

export type TilesetEditorRecord = {
  mode: "tileset";
  prompt: string;
  tileset: { gridSize: number; items: EditorTilesetItem[] };
};

export type UiEditorRecord = {
  mode: "ui";
  prompt: string;
  ui: { components: EditorUiComponent[] };
};

export type AudioEditorRecord = {
  mode: "audio";
  prompt: string;
  audio: Record<string, never>;
};

export type EditorRecord =
  | CharacterEditorRecord
  | SceneryEditorRecord
  | TilesetEditorRecord
  | UiEditorRecord
  | AudioEditorRecord;

export type EditorRecordForKind<K extends AssetKind> =
  K extends CharacterAssetKind
    ? CharacterEditorRecord
    : K extends SceneryAssetKind
      ? SceneryEditorRecord
      : K extends TilesetAssetKind
        ? TilesetEditorRecord
        : K extends UiAssetKind
          ? UiEditorRecord
          : AudioEditorRecord;

export function editorModeForAssetKind(kind: AssetKind): EditorRecord["mode"] {
  switch (kind) {
    case "character":
    case "object":
      return "character";
    case "scenery":
      return "scenery";
    case "tileset":
      return "tileset";
    case "ui":
      return "ui";
    case "audio":
      return "audio";
  }
}
