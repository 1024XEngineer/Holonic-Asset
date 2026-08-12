import type { AssetKind } from "../types";
import { z } from "zod";
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
      return "N/A";
    case "tileSet":
      return formatAssetSize({
        width:
          item.dimensions.tileSize.width * item.dimensions.tileAmount.columns,
        height:
          item.dimensions.tileSize.height * item.dimensions.tileAmount.rows,
      });
    default:
      return formatAssetSize(item.dimensions);
  }
}

const assetCanvasSizePattern = /^(\d+)\s*(?:×|x)\s*(\d+)\s*(?:px)?$/i;

export const assetCanvasSizeSchema = z
  .string()
  .trim()
  .min(1, "Canvas size is required.")
  .regex(
    assetCanvasSizePattern,
    "Canvas size must use a positive width × height value.",
  )
  .refine((value) => {
    const match = value.match(assetCanvasSizePattern);
    const width = Number(match?.[1]);
    const height = Number(match?.[2]);
    return (
      Number.isSafeInteger(width) &&
      width > 0 &&
      Number.isSafeInteger(height) &&
      height > 0
    );
  }, "Canvas size must use a positive width × height value.");

export const assetCanvasSizeDimensionsSchema = assetCanvasSizeSchema.transform(
  (value) => {
    const match = value.match(assetCanvasSizePattern);
    return { width: Number(match![1]), height: Number(match![2]) };
  },
);

function formatAssetSize({ width, height }: AssetSizeResponse) {
  return `${width} × ${height} px`;
}
