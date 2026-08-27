import { useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import {
  refreshAssetLibraryCacheInBackground,
  removeAssetFromLibraryCache,
  restoreAssetLibraryCache,
} from "./asset-library-cache";
import { assetKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";
import { useOptimisticDeleteMutation } from "@/model/optimistic-delete.mutation";

type DeleteAssetInput = { projectId: string; assetId: string };
const deleteAssetMutationKey = ["assets", "delete"] as const;

export function useDeleteAssetMutation() {
  const queryClient = useQueryClient();
  const userID = readAuthenticatedUserId();

  return useOptimisticDeleteMutation({
    mutationKey: deleteAssetMutationKey,
    mutationFn: ({ assetId }: DeleteAssetInput) => assetApi.delete(assetId),
    removeFromCache: async ({ projectId, assetId }) => {
      const queryKey = assetKeys.library(userID, projectId);
      await queryClient.cancelQueries({ queryKey });
      const snapshot = removeAssetFromLibraryCache(
        queryClient,
        userID,
        projectId,
        assetId,
      );
      return { projectId, snapshot };
    },
    restoreCache: ({ projectId }, context) => {
      restoreAssetLibraryCache(
        queryClient,
        userID,
        projectId,
        context.snapshot,
      );
    },
    isSameScope: (variables, { projectId }) =>
      variables?.projectId === projectId,
    refreshCache: ({ projectId }) => {
      refreshAssetLibraryCacheInBackground(queryClient, userID, projectId);
    },
  });
}
