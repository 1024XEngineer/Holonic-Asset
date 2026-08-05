import { getAssetKindConfig } from "@/components/asset-kind";
import type { AssetKind, ProjectAsset } from "@/model/asset";

import type { AssetLibraryItem } from "../types/asset";

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
