import { useMemo, useState } from "react";

import { assetKinds, type AssetGroup, type AssetKind } from "@/model/asset";

import { toAssetLibraryItem } from "../lib/to-asset-library-item";
import type { AssetLibraryItem } from "../types/asset";

export function filterAssetGroups(
  assetGroups: AssetGroup[],
  query: string,
  selectedKinds: AssetKind[],
): AssetLibraryItem[] {
  const normalizedQuery = query.trim().toLocaleLowerCase();

  return assetGroups
    .filter((group) => selectedKinds.includes(group.kind))
    .flatMap((group) => {
      return group.assets
        .filter((asset) => {
          if (!normalizedQuery) return true;

          return [
            asset.name,
            asset.description,
            asset.version,
            asset.canvasSize,
            asset.perspective,
            ...asset.tags,
          ].some((value) =>
            value.toLocaleLowerCase().includes(normalizedQuery),
          );
        })
        .map((asset) => toAssetLibraryItem(asset, group.kind));
    });
}

export function countAssetsByKind(assetGroups: AssetGroup[]) {
  const counts = Object.fromEntries(
    assetKinds.map((kind) => [kind, 0]),
  ) as Record<AssetKind, number>;

  for (const group of assetGroups) counts[group.kind] += group.assets.length;

  return counts;
}

export function useAssetLibrary(assetGroups: AssetGroup[], query: string) {
  const [selectedKinds, setSelectedKinds] = useState<AssetKind[]>([
    ...assetKinds,
  ]);
  const counts = useMemo(() => countAssetsByKind(assetGroups), [assetGroups]);
  const filteredAssets = useMemo(
    () => filterAssetGroups(assetGroups, query, selectedKinds),
    [assetGroups, query, selectedKinds],
  );
  const totalAssets = useMemo(
    () => Object.values(counts).reduce((total, count) => total + count, 0),
    [counts],
  );

  return {
    counts,
    filteredAssets,
    selectedKinds,
    setSelectedKinds,
    totalAssets,
  };
}
