import type { AssetKind } from "@/features/assets";
import type {
  EditorCanvasPosition,
  EditorCharacterAnimation,
  EditorCharacterAnimationClip,
  EditorCharacterSpriteSheet,
  EditorRecord,
  EditorRecordForKind,
  EditorSceneryLayer,
  EditorTilesetCell,
  EditorTilesetItem,
  EditorUiComponent,
} from "@/features/asset-editor";

export function isEditorRecordForAssetKind<K extends AssetKind>(
  kind: K,
  record: unknown,
): record is EditorRecordForKind<K> {
  return (
    isEditorRecord(record) && recordModeMatchesAssetKind(kind, record.mode)
  );
}

function recordModeMatchesAssetKind(
  kind: AssetKind,
  mode: EditorRecord["mode"],
) {
  return kind === "object" ? mode === "character" : mode === kind;
}

function isEditorRecord(value: unknown): value is EditorRecord {
  if (!isPlainObject(value) || typeof value.prompt !== "string") return false;

  switch (value.mode) {
    case "character":
      return (
        isPlainObject(value.character) &&
        isEditorCharacterSpriteSheet(value.character.prototype) &&
        isNodePositions(value.character.nodePositions) &&
        (value.character.animations === undefined ||
          isEditorCharacterAnimations(value.character.animations))
      );
    case "scenery":
      return (
        isPlainObject(value.scenery) &&
        isArrayOf(value.scenery.layers, isEditorSceneryLayer)
      );
    case "tileset":
      return (
        isPlainObject(value.tileset) &&
        isFiniteNumber(value.tileset.gridSize) &&
        isArrayOf(value.tileset.items, isEditorTilesetItem)
      );
    case "ui":
      return (
        isPlainObject(value.ui) &&
        isArrayOf(value.ui.components, isEditorUiComponent)
      );
    case "audio":
      return isPlainObject(value.audio);
    default:
      return false;
  }
}

function isNodePositions(
  value: unknown,
): value is Record<string, EditorCanvasPosition> {
  return (
    isPlainObject(value) && Object.values(value).every(isEditorCanvasPosition)
  );
}

function isEditorCanvasPosition(value: unknown): value is EditorCanvasPosition {
  return (
    isPlainObject(value) && isFiniteNumber(value.x) && isFiniteNumber(value.y)
  );
}

function isEditorCharacterAnimation(
  value: unknown,
): value is EditorCharacterAnimation {
  return isEditorCharacterAnimationClip(value);
}

function isEditorCharacterAnimationClip(
  value: unknown,
): value is EditorCharacterAnimationClip {
  return (
    isPlainObject(value) &&
    value.kind === "clip" &&
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
  return hasUniqueAnimationIds(value);
}

function hasUniqueAnimationIds(value: Array<{ id: string }>) {
  return new Set(value.map((animation) => animation.id)).size === value.length;
}

function isEditorCharacterAudio(
  value: unknown,
): value is NonNullable<EditorCharacterAnimationClip["audio"]> {
  return (
    isPlainObject(value) &&
    typeof value.label === "string" &&
    typeof value.time === "string"
  );
}

function isEditorCharacterSpriteSheet(
  value: unknown,
): value is EditorCharacterSpriteSheet {
  return (
    isPlainObject(value) &&
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
    isPlainObject(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    typeof value.detail === "string" &&
    typeof value.imageUrl === "string" &&
    (value.blendMode === "normal" || value.blendMode === "multiply")
  );
}

function isEditorTilesetItem(value: unknown): value is EditorTilesetItem {
  return (
    isPlainObject(value) &&
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
    isPlainObject(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    (value.kind === "panel" ||
      value.kind === "label" ||
      value.kind === "button") &&
    isPlainObject(value.bounds) &&
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

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
