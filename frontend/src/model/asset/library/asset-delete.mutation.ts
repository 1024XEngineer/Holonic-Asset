import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import {
  refreshAssetLibraryCacheInBackground,
  removeAssetFromLibraryCache,
  restoreAssetLibraryCache,
} from "./asset-library-cache";
import { assetKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

type DeleteAssetInput = { projectId: string; assetId: string };
const deleteAssetMutationKey = ["assets", "delete"] as const;

export function useDeleteAssetMutation() {
  const queryClient = useQueryClient();
  const userID = readAuthenticatedUserId();

  return useMutation({
    mutationKey: deleteAssetMutationKey,
    mutationFn: ({ assetId }: DeleteAssetInput) => assetApi.delete(assetId),
    onMutate: async ({ projectId, assetId }) => {
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
    onError: (_error, { projectId }, context) => {
      if (!context) return;
      restoreAssetLibraryCache(
        queryClient,
        userID,
        projectId,
        context.snapshot,
      );
    },
    onSettled: (_data, _error, { projectId }) => {
      if (
        queryClient.isMutating({
          mutationKey: deleteAssetMutationKey,
          predicate: (mutation) =>
            (mutation.state.variables as DeleteAssetInput | undefined)
              ?.projectId === projectId,
        }) !== 1
      ) {
        return;
      }
      refreshAssetLibraryCacheInBackground(queryClient, userID, projectId);
    },
  });
}
