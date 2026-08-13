import type { QueryClient } from "@tanstack/react-query";

import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";
import { createAssetLibraryCollection } from "./asset-library-collection";
import { readAuthenticatedUserId } from "@/model/auth";
import type { AssetKind, ProjectAsset } from "../types";

type ProjectScopedAssetMutation = { projectId: string };

export type AssetLibraryCacheSnapshot = {
  removed?: {
    asset: ProjectAsset;
    assetIndex: number;
    groupIndex: number;
    kind: AssetKind;
  };
};

export function removeAssetFromLibraryCache(
  queryClient: QueryClient,
  userID: number,
  projectId: string,
  assetId: string,
): AssetLibraryCacheSnapshot {
  const queryKey = assetKeys.library(userID, projectId);
  const current = queryClient.getQueryData<AssetGroup[]>(queryKey);
  if (!current) return {};

  const groupIndex = current.findIndex((group) =>
    group.assets.some((asset) => asset.id === assetId),
  );
  if (groupIndex < 0) return {};

  const group = current[groupIndex];
  const assetIndex = group.assets.findIndex((asset) => asset.id === assetId);
  const asset = group.assets[assetIndex];
  queryClient.setQueryData<AssetGroup[]>(
    queryKey,
    createAssetLibraryCollection(current, { projectId }).remove(assetId),
  );

  return {
    removed: { asset, assetIndex, groupIndex, kind: group.kind },
  };
}

export function restoreAssetLibraryCache(
  queryClient: QueryClient,
  userID: number,
  projectId: string,
  snapshot: AssetLibraryCacheSnapshot,
) {
  if (!snapshot.removed) return;
  const { asset, assetIndex, groupIndex, kind } = snapshot.removed;

  queryClient.setQueryData<AssetGroup[]>(
    assetKeys.library(userID, projectId),
    (current) => {
      if (!current) return current;
      const collection = createAssetLibraryCollection(current, { projectId });
      if (collection.find(asset.id)) return current;

      const currentGroupIndex = current.findIndex(
        (group) => group.kind === kind,
      );
      if (currentGroupIndex < 0) {
        const groups = [...current];
        groups.splice(groupIndex, 0, { kind, assets: [asset] });
        return groups;
      }

      return current.map((group, index) => {
        if (index !== currentGroupIndex) return group;
        const assets = [...group.assets];
        assets.splice(assetIndex, 0, asset);
        return { ...group, assets };
      });
    },
  );
}

export function refreshAssetLibraryCacheInBackground(
  queryClient: QueryClient,
  userID: number,
  projectId: string,
) {
  void refreshAssetLibraryCache(queryClient, userID, projectId).catch(
    () => undefined,
  );
}

export function refreshAssetLibraryCache(
  queryClient: QueryClient,
  userID: number,
  projectId: string,
) {
  return queryClient.refetchQueries({
    queryKey: assetKeys.library(userID, projectId),
    type: "all",
  });
}

export function createAssetLibraryCacheSync(
  queryClient: QueryClient,
  userID?: number,
) {
  return (
    assetGroups: AssetGroup[],
    { projectId }: ProjectScopedAssetMutation,
  ) => {
    queryClient.setQueryData<AssetGroup[]>(
      assetKeys.library(userID ?? readAuthenticatedUserId(), projectId),
      assetGroups,
    );
  };
}
