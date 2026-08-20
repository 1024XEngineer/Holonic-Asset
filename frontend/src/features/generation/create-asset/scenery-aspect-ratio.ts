export const sceneryDimensionsByAspectRatio = {
  "16:9": { width: 1536, height: 1024 },
  "4:3": { width: 1024, height: 768 },
  "21:9": { width: 1792, height: 768 },
  "1:1": { width: 1024, height: 1024 },
  "3:2": { width: 1536, height: 1024 },
  "9:16": { width: 1024, height: 1536 },
  "2:3": { width: 1024, height: 1536 },
} as const;

export type SceneryAspectRatio = keyof typeof sceneryDimensionsByAspectRatio;

export const sceneryAspectRatios = Object.keys(
  sceneryDimensionsByAspectRatio,
) as SceneryAspectRatio[];

export const defaultSceneryAspectRatio = "16:9" satisfies SceneryAspectRatio;

export function getSceneryDimensions(aspectRatio: SceneryAspectRatio) {
  return { ...sceneryDimensionsByAspectRatio[aspectRatio] };
}

export function getSceneryCanvasSize(aspectRatio: SceneryAspectRatio) {
  const { width, height } = getSceneryDimensions(aspectRatio);
  return `${width} × ${height} px`;
}
