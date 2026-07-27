import type { AssetKind, AssetRevision } from "@/features/assets/domain";

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

export type EditorSpriteSheetTile = {
  id: string;
  label: string;
  cells: number[];
};

export type EditorSpriteSheetItem = {
  id: string;
  label: string;
  icon: "bed" | "lamp" | "fence" | "object";
  tiles: EditorSpriteSheetTile[];
};

export type CharacterAssetKind = "character" | "object";
export type SceneryAssetKind = "scenery";
export type SpriteSheetAssetKind = Exclude<
  AssetKind,
  CharacterAssetKind | SceneryAssetKind
>;

export type CharacterEditorDocument = {
  mode: "character";
  prompt: string;
  character: {
    prototypeName?: string;
    animations?: EditorCharacterAnimation[];
    nodePositions: Record<string, EditorCanvasPosition>;
  };
};

export type SceneryEditorDocument = {
  mode: "scenery";
  prompt: string;
  scenery: { layers: EditorSceneryLayer[] };
};

export type SpriteSheetEditorDocument = {
  mode: "sprite-sheet";
  prompt: string;
  spriteSheet: { gridSize: number; items: EditorSpriteSheetItem[] };
};

export type EditorDocument =
  | CharacterEditorDocument
  | SceneryEditorDocument
  | SpriteSheetEditorDocument;

export type EditorDocumentForKind<K extends AssetKind> =
  K extends CharacterAssetKind
    ? CharacterEditorDocument
    : K extends SceneryAssetKind
      ? SceneryEditorDocument
      : SpriteSheetEditorDocument;

export function editorModeForAssetKind(
  kind: AssetKind,
): EditorDocument["mode"] {
  if (kind === "character" || kind === "object") return "character";
  if (kind === "scenery") return "scenery";
  return "sprite-sheet";
}

export function isEditorDocumentForAssetKind<K extends AssetKind>(
  kind: K,
  content: unknown,
): content is EditorDocumentForKind<K> {
  return (
    isEditorDocument(content) && content.mode === editorModeForAssetKind(kind)
  );
}

export type EditorWorkspaceAsset<K extends AssetKind = AssetKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  version: string;
  history: AssetRevision[];
};

export type EditorWorkspaceDataForKind<K extends AssetKind> = {
  projectName: string;
  asset: EditorWorkspaceAsset<K>;
  content: EditorDocumentForKind<K>;
};

export type EditorWorkspaceData = EditorWorkspaceDataForKind<AssetKind>;

function isEditorDocument(value: unknown): value is EditorDocument {
  if (!isRecord(value) || typeof value.prompt !== "string") return false;

  switch (value.mode) {
    case "character":
      return (
        isRecord(value.character) &&
        (value.character.prototypeName === undefined ||
          typeof value.character.prototypeName === "string") &&
        isNodePositions(value.character.nodePositions) &&
        (value.character.animations === undefined ||
          isArrayOf(value.character.animations, isEditorCharacterAnimation))
      );
    case "scenery":
      return (
        isRecord(value.scenery) &&
        isArrayOf(value.scenery.layers, isEditorSceneryLayer)
      );
    case "sprite-sheet":
      return (
        isRecord(value.spriteSheet) &&
        isFiniteNumber(value.spriteSheet.gridSize) &&
        isArrayOf(value.spriteSheet.items, isEditorSpriteSheetItem)
      );
    default:
      return false;
  }
}

function isNodePositions(
  value: unknown,
): value is Record<string, EditorCanvasPosition> {
  return isRecord(value) && Object.values(value).every(isEditorCanvasPosition);
}

function isEditorCanvasPosition(value: unknown): value is EditorCanvasPosition {
  return isRecord(value) && isFiniteNumber(value.x) && isFiniteNumber(value.y);
}

function isEditorCharacterAnimation(
  value: unknown,
): value is EditorCharacterAnimation {
  return (
    isRecord(value) &&
    isEditorCharacterAnimationId(value.id) &&
    typeof value.label === "string" &&
    isFiniteNumber(value.frameCount) &&
    (value.audio === undefined || isEditorCharacterAudio(value.audio))
  );
}

function isEditorCharacterAnimationId(
  value: unknown,
): value is EditorCharacterAnimationId {
  return (
    value === "idle" ||
    value === "walk" ||
    value === "harvest" ||
    value === "jump" ||
    value === "celebrate"
  );
}

function isEditorCharacterAudio(
  value: unknown,
): value is NonNullable<EditorCharacterAnimation["audio"]> {
  return (
    isRecord(value) &&
    typeof value.label === "string" &&
    typeof value.time === "string"
  );
}

function isEditorSceneryLayer(value: unknown): value is EditorSceneryLayer {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    typeof value.detail === "string" &&
    typeof value.imageUrl === "string" &&
    (value.blendMode === "normal" || value.blendMode === "multiply")
  );
}

function isEditorSpriteSheetItem(
  value: unknown,
): value is EditorSpriteSheetItem {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    (value.icon === "bed" ||
      value.icon === "lamp" ||
      value.icon === "fence" ||
      value.icon === "object") &&
    isArrayOf(value.tiles, isEditorSpriteSheetTile)
  );
}

function isEditorSpriteSheetTile(
  value: unknown,
): value is EditorSpriteSheetTile {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    isArrayOf(value.cells, isFiniteNumber)
  );
}

function isArrayOf<T>(
  value: unknown,
  guard: (entry: unknown) => entry is T,
): value is T[] {
  return Array.isArray(value) && value.every(guard);
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
