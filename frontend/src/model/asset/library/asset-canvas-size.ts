import type { AssetKind } from "../types";
import type {
  AssetListItemResponse,
  AssetSizeResponse,
} from "./asset.contract";
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

export function resolveAssetCanvasSize(item: AssetListItemResponse) {
  switch (item.type) {
    case "audio":
      return getDefaultAssetCanvasSize("audio");
    case "tileSet":
      return formatAssetSize(item.dimensions.tileSize);
    default:
      return formatAssetSize(item.dimensions);
  }
}

function formatAssetSize({ width, height }: AssetSizeResponse) {
  return `${width} × ${height} px`;
}
