export const assetKinds = [
  "character",
  "object",
  "tileset",
  "scenery",
  "audio",
  "ui",
] as const;

export type AssetKind = (typeof assetKinds)[number];

export type CreatableAssetKind = AssetKind;

export const creatableAssetKinds: CreatableAssetKind[] = [
  "character",
  "object",
  "tileset",
  "scenery",
  "ui",
  "audio",
];
