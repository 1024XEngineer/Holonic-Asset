import type { AssetKind } from "./asset-kind";

export type AssetRecordStatus = "ready" | "generating" | "failed";
export type EditorCanvasPosition = { x: number; y: number };
export type EditorCharacterAnimationId =
  | "idle"
  | "walk"
  | "harvest"
  | "jump"
  | "celebrate";
export type EditorCharacterAnimation = {
  id: EditorCharacterAnimationId;
  label: string;
  frameCount: number;
  audio?: { label: string; time: string };
};
export type EditorSceneryLayer = {
  id: string;
  label: string;
  detail: string;
  imageUrl: string;
  blendMode: "normal" | "multiply";
};
/** Global [column, row] coordinate in the tileset grid. */
export type EditorSpriteSheetCell = [column: number, row: number];
export type EditorSpriteSheetItem = {
  id: string;
  label: string;
  icon: "bed" | "lamp" | "fence" | "object";
  /**
   * The complete generated item image. Tiles are a selection/rendering map,
   * not independently stored images.
   */
  image?: { url: string; width: number; height: number };
  /** Every tileset grid cell occupied by this complete item. */
  cells: EditorSpriteSheetCell[];
};
export type CharacterAssetKind = "character" | "object";
export type SceneryAssetKind = "scenery";
export type SpriteSheetAssetKind = Exclude<
  AssetKind,
  CharacterAssetKind | SceneryAssetKind
>;
export type CharacterRecordContent = {
  mode: "character";
  prompt: string;
  character: {
    prototypeName?: string;
    animations?: EditorCharacterAnimation[];
    nodePositions: Record<string, EditorCanvasPosition>;
  };
};
export type SceneryRecordContent = {
  mode: "scenery";
  prompt: string;
  scenery: { layers: EditorSceneryLayer[] };
};
export type SpriteSheetRecordContent = {
  mode: "sprite-sheet";
  prompt: string;
  spriteSheet: { gridSize: number; items: EditorSpriteSheetItem[] };
};
export type RecordContent =
  | CharacterRecordContent
  | SceneryRecordContent
  | SpriteSheetRecordContent;
export type RecordContentForKind<K extends AssetKind> =
  K extends CharacterAssetKind
    ? CharacterRecordContent
    : K extends SceneryAssetKind
      ? SceneryRecordContent
      : SpriteSheetRecordContent;
export function recordModeForAssetKind(kind: AssetKind): RecordContent["mode"] {
  if (kind === "character" || kind === "object") return "character";
  if (kind === "scenery") return "scenery";
  return "sprite-sheet";
}
export function isRecordContentForAssetKind<K extends AssetKind>(
  kind: K,
  content: RecordContent,
): content is RecordContentForKind<K> {
  return content.mode === recordModeForAssetKind(kind);
}
export type AssetRecord = {
  id: string;
  version: string;
  description: string;
  savedAt?: string;
  status: AssetRecordStatus;
  isCurrent: boolean;
  content?: RecordContent;
};
export type RecordWorkspaceAsset<K extends AssetKind = AssetKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  version: string;
  history: AssetRecord[];
};
export type RecordDataForKind<K extends AssetKind> = {
  projectName: string;
  asset: RecordWorkspaceAsset<K>;
  content: RecordContentForKind<K>;
};
export type RecordData = RecordDataForKind<AssetKind>;
