import { DataApiError } from "@/lib/data-api-error";

import { assetKinds, type AssetKind, type ProjectAsset } from "../types";
import type { AssetGroup, AssetLibraryItem } from "./types";

type AssetLibraryScope = { projectId?: string };

type AssetLocation = {
  asset: ProjectAsset;
  assetIndex: number;
  groupIndex: number;
  item: AssetLibraryItem;
};

export type AssetLibraryCollection = {
  counts: Record<AssetKind, number>;
  items: readonly AssetLibraryItem[];
  totalAssets: number;
  append: (kind: AssetKind, asset: ProjectAsset) => AssetGroup[];
  find: (assetId: string) => AssetLibraryItem | undefined;
  insertAfter: (
    sourceAssetId: string,
    createAsset: (source: ProjectAsset) => ProjectAsset,
  ) => AssetGroup[];
  remove: (assetId: string) => AssetGroup[];
  update: (
    assetId: string,
    updateAsset: (asset: ProjectAsset) => ProjectAsset,
  ) => AssetGroup[];
};

export function createAssetLibraryCollection(
  groups: AssetGroup[],
  scope: AssetLibraryScope = {},
): AssetLibraryCollection {
  const counts = Object.fromEntries(
    assetKinds.map((kind) => [kind, 0]),
  ) as Record<AssetKind, number>;
  const locations = new Map<string, AssetLocation>();
  const items: AssetLibraryItem[] = [];

  groups.forEach((group, groupIndex) => {
    counts[group.kind] += group.assets.length;
    group.assets.forEach((asset, assetIndex) => {
      const item = { ...asset, kind: group.kind };
      items.push(item);
      if (!locations.has(asset.id)) {
        locations.set(asset.id, { asset, assetIndex, groupIndex, item });
      }
    });
  });

  function findLocation(assetId: string) {
    return locations.get(assetId);
  }

  function requireLocation(assetId: string) {
    const location = findLocation(assetId);
    if (location) return location;

    throw new DataApiError(
      "NOT_FOUND",
      "Asset was not found.",
      assetErrorDetails(scope, assetId),
    );
  }

  function requireAvailableId(assetId: string) {
    if (!findLocation(assetId)) return;

    throw new DataApiError(
      "CONFLICT",
      "Asset ID already exists.",
      assetErrorDetails(scope, assetId),
    );
  }

  function replaceGroupAssets(groupIndex: number, assets: ProjectAsset[]) {
    return groups.map((group, index) =>
      index === groupIndex ? { ...group, assets } : group,
    );
  }

  return {
    counts,
    items,
    totalAssets: items.length,
    append: (kind, asset) => {
      requireAvailableId(asset.id);
      const groupIndex = groups.findIndex((group) => group.kind === kind);
      if (groupIndex < 0) return [...groups, { kind, assets: [asset] }];

      return replaceGroupAssets(groupIndex, [
        ...groups[groupIndex].assets,
        asset,
      ]);
    },
    find: (assetId) => findLocation(assetId)?.item,
    insertAfter: (sourceAssetId, createAsset) => {
      const source = requireLocation(sourceAssetId);
      const asset = createAsset(source.asset);
      requireAvailableId(asset.id);
      const sourceGroup = groups[source.groupIndex];

      return replaceGroupAssets(source.groupIndex, [
        ...sourceGroup.assets.slice(0, source.assetIndex + 1),
        asset,
        ...sourceGroup.assets.slice(source.assetIndex + 1),
      ]);
    },
    remove: (assetId) => {
      const location = requireLocation(assetId);
      const group = groups[location.groupIndex];

      return replaceGroupAssets(location.groupIndex, [
        ...group.assets.slice(0, location.assetIndex),
        ...group.assets.slice(location.assetIndex + 1),
      ]);
    },
    update: (assetId, updateAsset) => {
      const location = requireLocation(assetId);
      const asset = updateAsset(location.asset);
      if (asset.id !== assetId) {
        throw new DataApiError(
          "BAD_REQUEST",
          "Asset updates cannot change asset identity.",
          assetErrorDetails(scope, assetId),
        );
      }
      const group = groups[location.groupIndex];
      const assets = [...group.assets];
      assets[location.assetIndex] = asset;
      return replaceGroupAssets(location.groupIndex, assets);
    },
  };
}

function assetErrorDetails(scope: AssetLibraryScope, assetId: string) {
  return scope.projectId
    ? { projectId: scope.projectId, assetId }
    : { assetId };
}
