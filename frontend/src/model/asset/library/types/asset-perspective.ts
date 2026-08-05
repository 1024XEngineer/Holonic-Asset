export const assetPerspectiveOptions = [
  "Top-down",
  "Side-On",
  "Isometric",
] as const;

export type AssetPerspective = (typeof assetPerspectiveOptions)[number];

export function isAssetPerspective(value: string): value is AssetPerspective {
  return assetPerspectiveOptions.includes(value as AssetPerspective);
}
