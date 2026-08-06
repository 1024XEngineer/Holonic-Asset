export const assetCanvasSizeOptions = [
  "16 × 16 px",
  "32 × 32 px",
  "48 × 64 px",
  "64 × 64 px",
  "128 × 128 px",
  "320 × 180 px",
  "1920 × 1080 px",
] as const;

export type AssetCanvasSize = (typeof assetCanvasSizeOptions)[number];
