import type {
  AssetKind,
  CharacterAnimation,
  CharacterAnimationClip,
  CharacterSpriteSheet,
  SceneryLayer,
} from "../types";
import type {
  AssetCanvasPosition,
  AssetRecord,
  AssetRecordForKind,
  TilesetCell,
  TilesetItem,
  UISetComponent,
} from "./types";
import type { SpriteAssetRecordData } from "./types/asset-record";

export function isAssetRecordForKind<K extends AssetKind>(
  kind: K,
  record: unknown,
): record is AssetRecordForKind<K> {
  return isAssetRecord(record) && record.mode === kind;
}

function isAssetRecord(value: unknown): value is AssetRecord {
  if (!isPlainObject(value) || typeof value.prompt !== "string") return false;

  switch (value.mode) {
    case "character":
      return isSpriteAssetRecordData(value.character);
    case "object":
      return isSpriteAssetRecordData(value.object);
    case "scenery":
      return (
        isPlainObject(value.scenery) &&
        isArrayOf(value.scenery.layers, isSceneryLayer)
      );
    case "tileset":
      return (
        isPlainObject(value.tileset) &&
        isFiniteNumber(value.tileset.gridSize) &&
        isArrayOf(value.tileset.items, isTilesetItem)
      );
    case "uiset":
      return (
        isPlainObject(value.uiset) &&
        isArrayOf(value.uiset.components, isUISetComponent)
      );
    case "audio":
      return isPlainObject(value.audio);
    default:
      return false;
  }
}

function isSpriteAssetRecordData(
  value: unknown,
): value is SpriteAssetRecordData {
  return (
    isPlainObject(value) &&
    isCharacterSpriteSheet(value.prototype) &&
    isNodePositions(value.nodePositions) &&
    (value.animations === undefined || isCharacterAnimations(value.animations))
  );
}

function isNodePositions(
  value: unknown,
): value is Record<string, AssetCanvasPosition> {
  return (
    isPlainObject(value) && Object.values(value).every(isAssetCanvasPosition)
  );
}

function isAssetCanvasPosition(value: unknown): value is AssetCanvasPosition {
  return (
    isPlainObject(value) && isFiniteNumber(value.x) && isFiniteNumber(value.y)
  );
}

function isCharacterAnimation(value: unknown): value is CharacterAnimation {
  return isCharacterAnimationClip(value);
}

function isCharacterAnimationClip(
  value: unknown,
): value is CharacterAnimationClip {
  return (
    isPlainObject(value) &&
    value.kind === "clip" &&
    typeof value.id === "string" &&
    value.id.length > 0 &&
    typeof value.label === "string" &&
    isPositiveInteger(value.frameCount) &&
    (value.spriteSheet === undefined ||
      isCharacterSpriteSheet(value.spriteSheet)) &&
    (value.audio === undefined || isCharacterAudio(value.audio))
  );
}

function isCharacterAnimations(value: unknown): value is CharacterAnimation[] {
  if (!isArrayOf(value, isCharacterAnimation)) return false;
  return hasUniqueAnimationIds(value);
}

function hasUniqueAnimationIds(value: Array<{ id: string }>) {
  return new Set(value.map((animation) => animation.id)).size === value.length;
}

function isCharacterAudio(
  value: unknown,
): value is NonNullable<CharacterAnimationClip["audio"]> {
  return (
    isPlainObject(value) &&
    typeof value.label === "string" &&
    typeof value.time === "string"
  );
}

function isCharacterSpriteSheet(value: unknown): value is CharacterSpriteSheet {
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

function isSceneryLayer(value: unknown): value is SceneryLayer {
  return (
    isPlainObject(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    typeof value.detail === "string" &&
    typeof value.imageUrl === "string" &&
    (value.blendMode === "normal" || value.blendMode === "multiply")
  );
}

function isTilesetItem(value: unknown): value is TilesetItem {
  return (
    isPlainObject(value) &&
    typeof value.id === "string" &&
    typeof value.label === "string" &&
    (value.imageUrl === undefined || typeof value.imageUrl === "string") &&
    isArrayOf(value.tiles, isTilesetCell)
  );
}

function isTilesetCell(value: unknown): value is TilesetCell {
  return (
    Array.isArray(value) &&
    value.length === 2 &&
    isFiniteNumber(value[0]) &&
    isFiniteNumber(value[1])
  );
}

function isUISetComponent(value: unknown): value is UISetComponent {
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
