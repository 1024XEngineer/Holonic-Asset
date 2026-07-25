export const assetKinds = [
  "character",
  "object",
  "tiles",
  "scenery",
  "audio",
  "background",
  "ui",
] as const;

export type AssetKind = (typeof assetKinds)[number];

export type CreatableAssetKind = AssetKind;

export const defaultAssetCanvasSize = "32 × 32 px";

const defaultCanvasSizeByAssetKind: Record<AssetKind, string> = {
  character: defaultAssetCanvasSize,
  object: defaultAssetCanvasSize,
  tiles: "16 × 16 px",
  scenery: defaultAssetCanvasSize,
  audio: defaultAssetCanvasSize,
  background: defaultAssetCanvasSize,
  ui: defaultAssetCanvasSize,
};

export function getDefaultAssetCanvasSize(kind: AssetKind) {
  return defaultCanvasSizeByAssetKind[kind];
}

export const creatableAssetKinds: CreatableAssetKind[] = [
  "character",
  "object",
  "background",
  "ui",
  "audio",
];
