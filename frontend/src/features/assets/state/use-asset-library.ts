import { useMemo, useState } from "react";

import {
  assetKinds,
  type AssetKind,
  type AssetLibraryCollection,
  type AssetLibraryItem,
} from "@/model/asset";

export function filterAssetLibraryItems(
  items: readonly AssetLibraryItem[],
  query: string,
  selectedKinds: readonly AssetKind[],
) {
  const normalizedQuery = query.trim().toLocaleLowerCase();

  return items.filter((item) => {
    if (!selectedKinds.includes(item.kind)) return false;
    if (!normalizedQuery) return true;

    return [
      item.name,
      item.description,
      item.version,
      item.canvasSize,
      item.perspective,
      ...item.tags,
    ].some((value) => value.toLocaleLowerCase().includes(normalizedQuery));
  });
}

export function useAssetLibrary(
  collection: AssetLibraryCollection,
  query: string,
) {
  const [selectedKinds, setSelectedKinds] = useState<AssetKind[]>([
    ...assetKinds,
  ]);
  const filteredAssets = useMemo(
    () => filterAssetLibraryItems(collection.items, query, selectedKinds),
    [collection, query, selectedKinds],
  );

  return {
    counts: collection.counts,
    filteredAssets,
    selectedKinds,
    setSelectedKinds,
    totalAssets: collection.totalAssets,
  };
}
