import type { AssetKind, AssetRevision } from "@/features/assets/domain";

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
  id: string;
  label: string;
  frameCount: number;
  spriteSheet?: EditorCharacterSpriteSheet;
  audio?: { label: string; time: string };
};

export type EditorCharacterAnimationGroup = {
  id: string;
  label: string;
  directions: EditorCharacterAnimationClip[];
};

export type EditorCharacterAnimation =
  | EditorCharacterAnimationClip
  | EditorCharacterAnimationGroup;

export function isEditorCharacterAnimationGroup(
  animation: EditorCharacterAnimation,
): animation is EditorCharacterAnimationGroup {
  return "directions" in animation;
}

export function getEditorCharacterAnimationClips(
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationClip[] {
  return animations.flatMap((animation) =>
    isEditorCharacterAnimationGroup(animation)
      ? animation.directions
      : [animation],
  );
}

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

export type CharacterEditorDocument = {
  mode: "character";
  prompt: string;
  character: {
    prototype: EditorCharacterSpriteSheet;
    animations?: EditorCharacterAnimation[];
    nodePositions: Record<string, EditorCanvasPosition>;
  };
};

export type SceneryEditorDocument = {
  mode: "scenery";
  prompt: string;
  scenery: { layers: EditorSceneryLayer[] };
};

export type TilesetEditorDocument = {
  mode: "tileset";
  prompt: string;
  tileset: { gridSize: number; items: EditorTilesetItem[] };
};

export type UiEditorDocument = {
  mode: "ui";
  prompt: string;
  ui: { components: EditorUiComponent[] };
};

export type AudioEditorDocument = {
  mode: "audio";
  prompt: string;
  audio: Record<string, never>;
};

export type EditorDocument =
  | CharacterEditorDocument
  | SceneryEditorDocument
  | TilesetEditorDocument
  | UiEditorDocument
  | AudioEditorDocument;

export type EditorDocumentForKind<K extends AssetKind> =
  K extends CharacterAssetKind
    ? CharacterEditorDocument
    : K extends SceneryAssetKind
      ? SceneryEditorDocument
      : K extends TilesetAssetKind
        ? TilesetEditorDocument
        : K extends UiAssetKind
          ? UiEditorDocument
          : AudioEditorDocument;

export function editorModeForAssetKind(
  kind: AssetKind,
): EditorDocument["mode"] {
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
        isEditorCharacterSpriteSheet(value.character.prototype) &&
        isNodePositions(value.character.nodePositions) &&
        (value.character.animations === undefined ||
          isEditorCharacterAnimations(value.character.animations))
      );
    case "scenery":
      return (
        isRecord(value.scenery) &&
        isArrayOf(value.scenery.layers, isEditorSceneryLayer)
      );
    case "tileset":
      return (
        isRecord(value.tileset) &&
        isFiniteNumber(value.tileset.gridSize) &&
        isArrayOf(value.tileset.items, isEditorTilesetItem)
      );
    case "ui":
      return (
        isRecord(value.ui) &&
        isArrayOf(value.ui.components, isEditorUiComponent)
      );
    case "audio":
      return isRecord(value.audio);
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
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    value.id.length === 0 ||
    typeof value.label !== "string"
  ) {
    return false;
  }

  if (value.directions !== undefined) {
    return (
      value.frameCount === undefined &&
      value.spriteSheet === undefined &&
      value.audio === undefined &&
      isArrayOf(value.directions, isEditorCharacterAnimationClip) &&
      value.directions.length > 0 &&
      value.directions.every((direction) =>
        direction.id.startsWith(`${value.id}/`),
      ) &&
      hasUniqueAnimationIds(value.directions)
    );
  }

  return isEditorCharacterAnimationClip(value);
}

function isEditorCharacterAnimationClip(
  value: unknown,
): value is EditorCharacterAnimationClip {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    value.id.length > 0 &&
    typeof value.label === "string" &&
    isPositiveInteger(value.frameCount) &&
    (value.spriteSheet === undefined ||
      isEditorCharacterSpriteSheet(value.spriteSheet)) &&
    (value.audio === undefined || isEditorCharacterAudio(value.audio))
  );
}

function isEditorCharacterAnimations(
  value: unknown,
): value is EditorCharacterAnimation[] {
  if (!isArrayOf(value, isEditorCharacterAnimation)) return false;
  return hasUniqueAnimationIds(
    value.flatMap((animation) =>
      "directions" in animation
        ? [animation, ...animation.directions]
        : [animation],
    ),
  );
}

function hasUniqueAnimationIds(value: Array<{ id: string }>) {
  return new Set(value.map((animation) => animation.id)).size === value.length;
}

function isEditorCharacterAudio(
  value: unknown,
): value is NonNullable<EditorCharacterAnimationClip["audio"]> {
  return (
    isRecord(value) &&
    typeof value.label === "string" &&
    typeof value.time === "string"
  );
}

function isEditorCharacterSpriteSheet(
  value: unknown,
): value is EditorCharacterSpriteSheet {
  return (
    isRecord(value) &&
    value.format === "png-sprite-sheet" &&
    typeof value.imageUrl === "string" &&
    isPositiveInteger(value.frameWidth) &&
    isPositiveInteger(value.frameHeight) &&
    isPositiveInteger(value.columns) &&
    isPositiveInteger(value.rows) &&
    (value.row === undefined ||
      (typeof value.row === "number" &&
        Number.isInteger(value.row) &&
        value.row >= 0 &&
        value.row < value.rows))
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

function isEditorTilesetItem(value: unknown): value is EditorTilesetItem {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    (value.imageUrl === undefined || typeof value.imageUrl === "string") &&
    isArrayOf(value.tiles, isEditorTilesetCell)
  );
}

function isEditorTilesetCell(value: unknown): value is EditorTilesetCell {
  return (
    Array.isArray(value) &&
    value.length === 2 &&
    isFiniteNumber(value[0]) &&
    isFiniteNumber(value[1])
  );
}

function isEditorUiComponent(value: unknown): value is EditorUiComponent {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    (value.kind === "panel" ||
      value.kind === "label" ||
      value.kind === "button") &&
    isRecord(value.bounds) &&
    isFiniteNumber(value.bounds.x) &&
    isFiniteNumber(value.bounds.y) &&
    isFiniteNumber(value.bounds.width) &&
    isFiniteNumber(value.bounds.height)
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

function isPositiveInteger(value: unknown): value is number {
  return typeof value === "number" && Number.isInteger(value) && value > 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
