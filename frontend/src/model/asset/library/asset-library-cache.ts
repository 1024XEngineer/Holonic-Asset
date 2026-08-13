import type { QueryClient } from "@tanstack/react-query";

import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";
import { readAuthenticatedUserId } from "@/model/auth";

type ProjectScopedAssetMutation = { projectId: string };

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
