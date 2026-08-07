import type { AssetKind } from "../types";
import {
  assetCanvasSizeOptions,
  type AssetCanvasSize,
} from "./types/asset-canvas-size";

export const defaultAssetCanvasSize = assetCanvasSizeOptions[1];

const defaultCanvasSizeByAssetKind: Record<AssetKind, AssetCanvasSize> = {
  character: defaultAssetCanvasSize,
  object: defaultAssetCanvasSize,
  tileset: "16 × 16 px",
  scenery: defaultAssetCanvasSize,
  audio: defaultAssetCanvasSize,
  uiset: defaultAssetCanvasSize,
};

export function getDefaultAssetCanvasSize(kind: AssetKind) {
  return defaultCanvasSizeByAssetKind[kind];
}
