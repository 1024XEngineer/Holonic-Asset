export const assetKinds = [
  "character",
  "object",
  "tileset",
  "scenery",
  "uiset",
  "audio",
] as const;

export type AssetKind = (typeof assetKinds)[number];

export type CreatableAssetKind = AssetKind;

export const creatableAssetKinds: CreatableAssetKind[] = [...assetKinds];
