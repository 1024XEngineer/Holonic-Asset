import type { QueryClient } from "@tanstack/react-query";

import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";

type ProjectScopedAssetMutation = { projectId: string };

export function createAssetLibraryCacheSync(queryClient: QueryClient) {
  return (
    assetGroups: AssetGroup[],
    { projectId }: ProjectScopedAssetMutation,
  ) => {
    queryClient.setQueryData<AssetGroup[]>(
      assetKeys.library(projectId),
      assetGroups,
    );
  };
}
