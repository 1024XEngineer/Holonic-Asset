import type { AssetKind } from "@/features/assets";

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
