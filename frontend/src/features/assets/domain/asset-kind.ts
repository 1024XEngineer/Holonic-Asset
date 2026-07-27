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

export const defaultAssetCanvasSize = "32 × 32 px";

const defaultCanvasSizeByAssetKind: Record<AssetKind, string> = {
  character: defaultAssetCanvasSize,
  object: defaultAssetCanvasSize,
  tileset: "16 × 16 px",
  scenery: defaultAssetCanvasSize,
  audio: defaultAssetCanvasSize,
  ui: defaultAssetCanvasSize,
};

export function getDefaultAssetCanvasSize(kind: AssetKind) {
  return defaultCanvasSizeByAssetKind[kind];
}

export const creatableAssetKinds: CreatableAssetKind[] = [
  "character",
  "object",
  "tileset",
  "scenery",
  "ui",
  "audio",
];
