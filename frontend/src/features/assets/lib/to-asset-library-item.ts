import { getAssetKindConfig } from "../asset-kind-config";
import type { AssetLibraryItem } from "../types/asset";
import type { AssetKind, ProjectAsset } from "@/model/asset";

export function toAssetLibraryItem(
  asset: ProjectAsset,
  kind: AssetKind,
): AssetLibraryItem {
  const config = getAssetKindConfig(kind);

  return {
    ...asset,
    kind,
    accentClassName: config.accentClassName,
    kindLabel: config.label,
  };
}
