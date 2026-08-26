export type {
  AssetCanvasPosition,
  AssetContent,
  AssetContentForKind,
  AssetContentKind,
  CharacterAssetContent,
  CharacterAssetKind,
  ObjectAssetContent,
  SceneryAssetContent,
  SceneryAssetKind,
  SceneryCanvasDimensions,
  SpriteAssetRecordData,
  TilesetAssetContent,
  TilesetAssetKind,
  TilesetItem,
  UISetAssetContent,
  UISetAssetKind,
  UISetComponent,
} from "./asset-content";

// Compatibility aliases for editor integrations that still use the old name.
export type {
  AssetContent as AssetRecord,
  AssetContentForKind as AssetRecordForKind,
  AssetContentKind as AssetRecordKind,
  CharacterAssetContent as CharacterAssetRecord,
  ObjectAssetContent as ObjectAssetRecord,
  SceneryAssetContent as SceneryAssetRecord,
  TilesetAssetContent as TilesetAssetRecord,
  UISetAssetContent as UISetAssetRecord,
} from "./asset-content";
