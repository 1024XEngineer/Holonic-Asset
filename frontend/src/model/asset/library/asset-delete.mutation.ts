import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { createAssetLibraryCacheSync } from "./asset-library-cache";

type DeleteAssetInput = { projectId: string; assetId: string };

export function useDeleteAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId }: DeleteAssetInput) =>
      assetApi.delete(projectId, assetId),
    onSuccess: createAssetLibraryCacheSync(queryClient),
  });
}
